package main

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
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

    // MongoDB Atlas
    mongoClient, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
    if err != nil {
        log.Fatalf("mongo connect: %v", err)
    }
    db := mongoClient.Database(cfg.MongoDB)

    // Redis
    rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})

    // RabbitMQ 
    amqpConn, err := amqp.Dial(cfg.RabbitMQURL)
    if err != nil {
        log.Fatalf("rabbitmq connect: %v", err)
    }

    // Firebase Admin SDK
    firebaseApp, err := firebase.NewApp(ctx, nil)
    if err != nil {
        log.Fatalf("firebase init: %v", err)
    }
    authClient, err := firebaseApp.Auth(ctx)
    if err != nil {
        log.Fatalf("firebase auth: %v", err)
    }

    // Router
    router := gin.Default()
    router.GET("/ping", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "pong"})
    })

    // Swagger UI — http://localhost:8080/swagger/index.html
    router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    api := router.Group("/api")
    {
        api.GET("/showtimes/:showtime_id/seats", handler.GetSeats)
        api.POST("/showtimes/:showtime_id/seats/:seat_id/lock", handler.LockSeat)
        api.POST("/showtimes/:showtime_id/seats/:seat_id/book", handler.ConfirmBooking)
    }

    log.Printf("server starting on :%s", cfg.Port)
    if err := router.Run(":" + cfg.Port); err != nil {
        log.Fatalf("server: %v", err)
    }

    _ = db
    _ = rdb
    _ = amqpConn
    _ = authClient
}