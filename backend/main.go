package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type URLData struct {
	ShortURL string `bson:"shorturl" json:"shorturl"`
	LongURL  string `bson:"longurl" json:"longurl"`
}

type ShortenRequest struct {
	LongURL string `json:"longurl"`
}

type URLStore interface {
	Save(context.Context, URLData) (bool, error)
	FindByShortURL(context.Context, string) (URLData, error)
}

const (
	base62Alphabet      = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	maxCollisionRetries = 100
	databaseTimeout     = 3 * time.Second
)

func createShortURL(longURL string, salt int) string {
	hashInput := longURL
	if salt > 0 {
		hashInput += ":" + strconv.Itoa(salt)
	}

	hash := sha256.Sum256([]byte(hashInput))
	value := new(big.Int).SetBytes(hash[:])
	base := big.NewInt(62)
	remainder := new(big.Int)
	shortURL := make([]byte, 6)

	for i := len(shortURL) - 1; i >= 0; i-- {
		value.QuoRem(value, base, remainder)
		shortURL[i] = base62Alphabet[remainder.Int64()]
	}

	return string(shortURL)
}

func validateLongURL(rawURL string) (string, bool) {
	trimmedURL := strings.TrimSpace(rawURL)
	lowerURL := strings.ToLower(trimmedURL)
	if !strings.Contains(trimmedURL, "://") {
		if strings.HasPrefix(lowerURL, "http:") || strings.HasPrefix(lowerURL, "https:") {
			return "", false
		}
		trimmedURL = "https://" + trimmedURL
	}

	parsedURL, err := url.Parse(trimmedURL)
	if err != nil {
		return "", false
	}
	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", false
	}
	if parsedURL.Hostname() == "" {
		return "", false
	}
	parsedURL.Host = strings.ToLower(parsedURL.Host)
	return parsedURL.String(), true
}

func shorten(store URLStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request ShortenRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "A valid long URL must be provided"})
			return
		}

		longURL, valid := validateLongURL(request.LongURL)
		if !valid {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "A valid HTTP or HTTPS URL must be provided"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), databaseTimeout)
		defer cancel()

		for salt := 0; salt < maxCollisionRetries; salt++ {
			newURL := URLData{
				ShortURL: createShortURL(longURL, salt),
				LongURL:  longURL,
			}

			created, err := store.Save(ctx, newURL)
			if errors.Is(err, ErrShortURLCollision) {
				continue
			}
			if err != nil {
				c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Could not save URL"})
				return
			}

			status := http.StatusCreated
			if !created {
				status = http.StatusOK
			}
			c.IndentedJSON(status, newURL)
			return
		}

		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Could not generate a unique short URL"})
	}
}

func resolveLongURL(store URLStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), databaseTimeout)
		defer cancel()

		url, err := store.FindByShortURL(ctx, c.Param("shorturl"))
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Short URL not found"})
			return
		}
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Could not retrieve URL"})
			return
		}

		c.Header("Cache-Control", "no-store")
		c.IndentedJSON(http.StatusOK, gin.H{"longurl": url.LongURL})
	}
}

func newRouter(store URLStore) *gin.Engine {
	router := gin.Default()
	router.POST("/shorten", shorten(store))
	router.GET("/resolve/:shorturl", resolveLongURL(store))
	return router
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	mongoURI := envOrDefault("MONGODB_URI", "mongodb://localhost:27017")
	databaseName := envOrDefault("MONGODB_DATABASE", "url_shortener")
	serverAddress := envOrDefault("SERVER_ADDRESS", "localhost:8080")

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
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("connect to MongoDB: %v", err)
	}

	store := NewMongoStore(client.Database(databaseName).Collection("urls"))
	if err := store.CreateIndexes(ctx); err != nil {
		log.Fatalf("create MongoDB indexes: %v", err)
	}

	if err := newRouter(store).Run(serverAddress); err != nil {
		log.Fatal(err)
	}
}
