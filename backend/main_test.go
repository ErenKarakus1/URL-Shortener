package main

import "testing"

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
			name:      "rejects missing scheme",
			input:     "example.com",
			wantValid: false,
		},
		{
			name:      "rejects unsupported scheme",
			input:     "ftp://example.com",
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
