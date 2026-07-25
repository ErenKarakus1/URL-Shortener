package main

import (
	"crypto/sha256"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
)

type URLData struct {
	ShortURL string `json:"shorturl"`
	LongURL  string `json:"longurl"`
}

type ShortenRequest struct {
	LongURL string `json:"longurl"`
}

var urls = []URLData{
	{ShortURL: "123456", LongURL: "google.com"},
	{ShortURL: "789123", LongURL: "youtube.com"},
}

const base62Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func createShortURL(longURL string) string {
	hash := sha256.Sum256([]byte(longURL))
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

func getUrls(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, urls)
}

func shorten(c *gin.Context) {
	var long ShortenRequest
	if err := c.BindJSON(&long); err != nil {
		c.IndentedJSON(http.StatusNotAcceptable, gin.H{"message": "Long URL must be provided"})
		return
	}
	var newURL URLData
	newURL.ShortURL = createShortURL(long.LongURL)
	newURL.LongURL = long.LongURL
	urls = append(urls, newURL)
	c.IndentedJSON(http.StatusCreated, newURL)
}

func returnLongUrl(c *gin.Context) {
	short := c.Param("shorturl")
	for _, url := range urls {
		if url.ShortURL == short {
			c.IndentedJSON(http.StatusOK, url.LongURL)
			return
		}
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Short URL not found"})
}

func main() {
	router := gin.Default()
	router.GET("/urls", getUrls)
	router.POST("/shorten", shorten)
	router.GET("/:shorturl", returnLongUrl)
	router.Run("localhost:8080")
}
