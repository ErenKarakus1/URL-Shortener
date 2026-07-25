package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type fakeStore struct {
	urls    map[string]URLData
	saveErr error
	findErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{urls: make(map[string]URLData)}
}

func (s *fakeStore) Save(_ context.Context, url URLData) (bool, error) {
	if s.saveErr != nil {
		return false, s.saveErr
	}
	if existing, found := s.urls[url.ShortURL]; found {
		if existing.LongURL == url.LongURL {
			return false, nil
		}
		return false, ErrShortURLCollision
	}
	s.urls[url.ShortURL] = url
	return true, nil
}

func (s *fakeStore) FindByShortURL(_ context.Context, shortURL string) (URLData, error) {
	if s.findErr != nil {
		return URLData{}, s.findErr
	}
	url, found := s.urls[shortURL]
	if !found {
		return URLData{}, mongo.ErrNoDocuments
	}
	return url, nil
}

func performRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestValidateLongURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantURL   string
		wantValid bool
	}{
		{
			name:      "valid HTTPS URL",
			input:     "https://example.com/path",
			wantURL:   "https://example.com/path",
			wantValid: true,
		},
		{
			name:      "trims surrounding spaces",
			input:     "  https://example.com/path  ",
			wantURL:   "https://example.com/path",
			wantValid: true,
		},
		{
			name:      "normalizes scheme and hostname",
			input:     "HTTPS://EXAMPLE.COM/CaseSensitive?Name=Eren#Section",
			wantURL:   "https://example.com/CaseSensitive?Name=Eren#Section",
			wantValid: true,
		},
		{
			name:      "adds HTTPS to missing scheme",
			input:     "example.com",
			wantURL:   "https://example.com",
			wantValid: true,
		},
		{
			name:      "adds HTTPS to domain with path",
			input:     "www.example.com/CaseSensitive",
			wantURL:   "https://www.example.com/CaseSensitive",
			wantValid: true,
		},
		{
			name:      "rejects unsupported scheme",
			input:     "ftp://example.com",
			wantValid: false,
		},
		{
			name:      "rejects malformed HTTP scheme",
			input:     "https:/example.com",
			wantValid: false,
		},
		{
			name:      "rejects missing hostname",
			input:     "https:///path",
			wantValid: false,
		},
		{
			name:      "rejects malformed URL",
			input:     "https://example.com/%zz",
			wantValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotURL, gotValid := validateLongURL(test.input)
			if gotURL != test.wantURL || gotValid != test.wantValid {
				t.Fatalf(
					"validateLongURL(%q) = (%q, %v), want (%q, %v)",
					test.input,
					gotURL,
					gotValid,
					test.wantURL,
					test.wantValid,
				)
			}
		})
	}
}

func TestCreateShortURLUsesTrimmedURL(t *testing.T) {
	trimmedURL, valid := validateLongURL("  https://example.com  ")
	if !valid {
		t.Fatal("expected URL to be valid")
	}

	if createShortURL(trimmedURL, 0) != createShortURL("https://example.com", 0) {
		t.Fatal("expected surrounding spaces not to affect the short URL")
	}
}

func TestCreateShortURLWithSalt(t *testing.T) {
	longURL := "https://example.com"

	withoutSalt := createShortURL(longURL, 0)
	withSalt := createShortURL(longURL, 1)

	if withoutSalt == withSalt {
		t.Fatal("expected salt to produce a different short URL")
	}
	if withSalt != createShortURL(longURL, 1) {
		t.Fatal("expected the same salt to produce the same short URL")
	}
}

func TestShortenAndReturnSameURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newRouter(newFakeStore())
	body := `{"longurl":"  HTTPS://EXAMPLE.COM/Path  "}`

	first := performRequest(router, http.MethodPost, "/shorten", body)
	if first.Code != http.StatusCreated {
		t.Fatalf("first request status = %d, want %d", first.Code, http.StatusCreated)
	}

	var firstURL URLData
	if err := json.Unmarshal(first.Body.Bytes(), &firstURL); err != nil {
		t.Fatal(err)
	}
	if firstURL.LongURL != "https://example.com/Path" || len(firstURL.ShortURL) != 6 {
		t.Fatalf("unexpected shortened URL: %+v", firstURL)
	}

	second := performRequest(router, http.MethodPost, "/shorten", body)
	if second.Code != http.StatusOK {
		t.Fatalf("second request status = %d, want %d", second.Code, http.StatusOK)
	}

	var secondURL URLData
	if err := json.Unmarshal(second.Body.Bytes(), &secondURL); err != nil {
		t.Fatal(err)
	}
	if secondURL != firstURL {
		t.Fatalf("same long URL produced %+v and %+v", firstURL, secondURL)
	}
}

func TestShortenRejectsInvalidURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newRouter(newFakeStore())

	response := performRequest(router, http.MethodPost, "/shorten", `{"longurl":"ftp://example.com"}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestShortenRetriesCollisionWithSalt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newFakeStore()
	longURL := "https://example.com"
	firstCode := createShortURL(longURL, 0)
	store.urls[firstCode] = URLData{ShortURL: firstCode, LongURL: "https://different.example"}
	router := newRouter(store)

	response := performRequest(router, http.MethodPost, "/shorten", `{"longurl":"https://example.com"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	var result URLData
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ShortURL != createShortURL(longURL, 1) {
		t.Fatalf("short URL = %q, want salted code %q", result.ShortURL, createShortURL(longURL, 1))
	}
}

func TestReturnLongURLRedirects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newFakeStore()
	store.urls["abc123"] = URLData{ShortURL: "abc123", LongURL: "https://example.com/path"}
	router := newRouter(store)

	response := performRequest(router, http.MethodGet, "/abc123", "")
	if response.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMovedPermanently)
	}
	if location := response.Header().Get("Location"); location != "https://example.com/path" {
		t.Fatalf("Location = %q, want %q", location, "https://example.com/path")
	}
}

func TestReturnLongURLNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newRouter(newFakeStore())

	response := performRequest(router, http.MethodGet, "/missing", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestDatabaseErrorsReturnInternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	databaseError := errors.New("database unavailable")

	saveStore := newFakeStore()
	saveStore.saveErr = databaseError
	saveResponse := performRequest(
		newRouter(saveStore),
		http.MethodPost,
		"/shorten",
		`{"longurl":"https://example.com"}`,
	)
	if saveResponse.Code != http.StatusInternalServerError {
		t.Fatalf("save status = %d, want %d", saveResponse.Code, http.StatusInternalServerError)
	}

	findStore := newFakeStore()
	findStore.findErr = databaseError
	findResponse := performRequest(newRouter(findStore), http.MethodGet, "/abc123", "")
	if findResponse.Code != http.StatusInternalServerError {
		t.Fatalf("find status = %d, want %d", findResponse.Code, http.StatusInternalServerError)
	}
}
