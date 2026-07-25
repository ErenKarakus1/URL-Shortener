package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type URLStore interface {
	Save(context.Context, URLData) (bool, error)
	FindByShortURL(context.Context, string) (URLData, error)
}

const (
	databaseTimeout     = 3 * time.Second
	maxRequestBodyBytes = 4 * 1024
)

func shorten(store URLStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)

		var request ShortenRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			var sizeError *http.MaxBytesError
			if errors.As(err, &sizeError) {
				c.IndentedJSON(http.StatusRequestEntityTooLarge, gin.H{"message": "Request body is too large"})
				return
			}
			c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "A valid long URL must be provided"})
			return
		}
		if len(strings.TrimSpace(request.LongURL)) > maxLongURLLength {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Long URL is too long"})
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

func newRouter(store URLStore, limiters ...RateLimiter) *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware(os.Getenv("CORS_ALLOWED_ORIGIN")))
	router.GET("/health", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"status": "ok"})
	})

	shortenHandlers := []gin.HandlerFunc{}
	if len(limiters) > 0 && limiters[0] != nil {
		shortenHandlers = append(shortenHandlers, rateLimitMiddleware(limiters[0]))
	}
	shortenHandlers = append(shortenHandlers, shorten(store))
	router.POST("/shorten", shortenHandlers...)
	router.GET("/resolve/:shorturl", resolveLongURL(store))
	return router
}
