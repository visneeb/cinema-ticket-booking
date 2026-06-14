package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	firebase "firebase.google.com/go/v4"
	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	_ "cinema-ticket-booking/docs"
	"cinema-ticket-booking/internal/config"
	"cinema-ticket-booking/internal/handler"
	"cinema-ticket-booking/internal/middleware"
	"cinema-ticket-booking/internal/model"
	"cinema-ticket-booking/internal/repository"
	"cinema-ticket-booking/internal/service"
	wshub "cinema-ticket-booking/internal/websocket"
	"cinema-ticket-booking/pkg/rabbitmq"
)

// @title           Cinema Ticket Booking API
// @version         1.0
// @description     Cinema seat booking API
// @host            localhost:8080
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()
	ctx := context.Background()

	db := connectMongo(cfg)
	rdb := connectRedis(cfg)
	mq := connectRabbitMQ(cfg)
	authCl := initFirebase(ctx)

	pub, err := rabbitmq.NewPublisher(mq)
	if err != nil {
		log.Fatalf("rabbitmq publisher: %v", err)
	}

	hub := wshub.NewHub()
	go hub.Run()

	lockRepo    := repository.NewSeatLockRepository(rdb)
	bookingRepo := repository.NewBookingRepository(db)
	adminRepo   := repository.NewRepository(db)
	userRepo    := repository.NewUserRepository(db)

	svc := service.NewBookingService(lockRepo, bookingRepo, pub, hub)
	for name, fn := range map[string]func(*amqp.Connection) error{
		"audit-timeout":    svc.StartAuditConsumer,
		"audit-log":        svc.StartAuditLogConsumer,
		"notification":     svc.StartNotificationConsumer,
	} {
		if err := fn(mq); err != nil {
			log.Printf("%s consumer: %v", name, err)
		}
	}

	adminSvc := service.NewAdminService(bookingRepo, adminRepo)
	userSvc  := service.NewUserService(userRepo)

	h := handler.New(authCl, hub, handler.Services{
		Booking: svc,
		User:    userSvc,
		Admin:   adminSvc,
	})
	router := setupRouter(h, userRepo)
	runServer(router, cfg.Port)
}

func connectMongo(cfg *config.Config) *mongo.Database {
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	return client.Database(cfg.MongoDB)
}

func connectRedis(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
}

func connectRabbitMQ(cfg *config.Config) *amqp.Connection {
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("rabbitmq connect: %v", err)
	}
	return conn
}

func initFirebase(ctx context.Context) *firebaseAuth.Client {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Fatalf("firebase init: %v", err)
	}
	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("firebase auth: %v", err)
	}
	return authClient
}

func setupRouter(h *handler.Handler, userRepo *repository.UserRepository) *gin.Engine {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// Swagger UI — http://localhost:8080/swagger/index.html
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// WebSocket — no auth middleware (token can be sent as query param if needed)
	router.GET("/ws/showtimes/:showtime_id", h.ServeWS)

	api := router.Group("/api")
	{
		// Users
		api.POST("/users/me", middleware.Auth(h.AuthCl), h.UpsertUser)

		// Showtimes & Bookings
		api.GET("/showtimes", h.GetShowtimes)
		api.GET("/showtimes/:showtime_id/seats", h.GetSeats)
		api.GET("/showtimes/:showtime_id/my-lock", middleware.Auth(h.AuthCl), h.GetMyLock)
		api.POST("/showtimes/:showtime_id/seats/:seat_id/lock", middleware.Auth(h.AuthCl), h.LockSeat)
		api.DELETE("/showtimes/:showtime_id/seats/:seat_id/lock", middleware.Auth(h.AuthCl), h.ReleaseLock)
		api.POST("/showtimes/:showtime_id/seats/:seat_id/book", middleware.Auth(h.AuthCl), h.ConfirmBooking)

		// Admin — requires authentication + admin role
		admin := api.Group("/admin",
			middleware.Auth(h.AuthCl),
			middleware.RequireRole(userRepo.FindByUID, model.RoleAdmin),
		)
		{
			admin.GET("/bookings", h.ListBookings)
			admin.GET("/movies",   h.ListMovies)
			admin.GET("/users",    h.ListUsers)
		}
	}

	return router
}

func runServer(router *gin.Engine, port string) {
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("server starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced shutdown: %v", err)
	}
	log.Println("server exited")
}