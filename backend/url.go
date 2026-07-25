package main

import (
	"crypto/sha256"
	"math/big"
	"net/url"
	"strconv"
	"strings"
)

type URLData struct {
	ShortURL string `bson:"shorturl" json:"shorturl"`
	LongURL  string `bson:"longurl" json:"longurl"`
}

type ShortenRequest struct {
	LongURL string `json:"longurl"`
}

const (
	base62Alphabet      = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	maxCollisionRetries = 100
	maxLongURLLength    = 2048
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
