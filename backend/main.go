package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func main() {
	mongoURI := envOrDefault("MONGODB_URI", "mongodb://localhost:27017")
	databaseName := envOrDefault("MONGODB_DATABASE", "url_shortener")
	redisAddress := envOrDefault("REDIS_ADDRESS", "localhost:6379")
	serverAddress := envOrDefault("SERVER_ADDRESS", "localhost:8080")
	rateLimitRequests := envIntOrDefault("RATE_LIMIT_REQUESTS", 30)
	rateLimitWindow := time.Duration(envIntOrDefault("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second

	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("disconnect MongoDB: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("connect to MongoDB: %v", err)
	}

	store := NewMongoStore(client.Database(databaseName).Collection("urls"))
	if err := store.CreateIndexes(ctx); err != nil {
		log.Fatalf("create MongoDB indexes: %v", err)
	}
	cancel()

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddress})
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("disconnect Redis: %v", err)
		}
	}()
	redisContext, cancelRedis := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(redisContext).Err(); err != nil {
		log.Fatalf("connect to Redis: %v", err)
	}
	cancelRedis()

	limiter := NewRedisRateLimiter(redisClient, rateLimitRequests, rateLimitWindow)
	if err := newRouter(store, limiter).Run(serverAddress); err != nil {
		log.Fatal(err)
	}
}
