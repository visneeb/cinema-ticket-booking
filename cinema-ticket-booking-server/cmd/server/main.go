package main

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"cinema-ticket-booking/internal/config"
)

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

    // TODO: wire middleware + handlers using db, rdb, amqpConn, authClient

    log.Printf("server starting on :%s", cfg.Port)
    if err := router.Run(":" + cfg.Port); err != nil {
        log.Fatalf("server: %v", err)
    }

    _ = db
    _ = rdb
    _ = amqpConn
    _ = authClient
}