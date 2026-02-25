package crawler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// snippet returns the first n characters of a string for logging
func snippet(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n] + "..."
}

func TestGetPageData(t *testing.T) {
	t.Run("Mock server tests", func(t *testing.T) {
		tests := []struct {
			name         string
			responseCode int
			responseBody string
			contentType  string
			wantErr      bool
		}{
			{
				name:         "HTML success",
				responseCode: 200,
				responseBody: "<html><body>Hello</body></html>",
				contentType:  "text/html; charset=utf-8",
				wantErr:      false,
			},
			{
				name:         "Non-HTML content type",
				responseCode: 200,
				responseBody: "Just plain text",
				contentType:  "text/plain",
				wantErr:      true,
			},
			{
				name:         "HTTP error 404",
				responseCode: 404,
				responseBody: "<html>Not found</html>",
				contentType:  "text/html",
				wantErr:      true,
			},
			{
				name:         "Empty body with 200 OK",
				responseCode: 200,
				responseBody: "",
				contentType:  "text/html",
				wantErr:      false,
			},
		}

		for _, tt := range tests {
			tt := tt // capture range variable
			t.Run(tt.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", tt.contentType)
					w.WriteHeader(tt.responseCode)
					w.Write([]byte(tt.responseBody))
				}))
				defer server.Close()

				body, statusCode, contentType, err := getPageData(server.URL)

				if (err != nil) != tt.wantErr {
					t.Fatalf("expected error: %v, got: %v, err: %v", tt.wantErr, err != nil, err)
				}

				fmt.Println("Test:", tt.name)
				fmt.Println("Status Code:", statusCode)
				fmt.Println("Content-Type:", contentType)
				fmt.Println("Body snippet:", snippet(body, 50))
				fmt.Println("Error:", err)
				fmt.Println("----------")

				if !tt.wantErr && body != tt.responseBody {
					t.Errorf("expected body: %q, got: %q", tt.responseBody, body)
				}
			})
		}
	})

	t.Run("Live URL test", func(t *testing.T) {
		url := "https://www.youtube.com"
		fmt.Println("=== Starting live URL test:", url, "===")
		body, statusCode, contentType, err := getPageData(url)

		if err != nil {
			t.Errorf("Error fetching URL: %v", err)
		} else {
			fmt.Println("Status Code:", statusCode)
			fmt.Println("Content-Type:", contentType)
			fmt.Println("Body snippet:", snippet(body, 200))
			t.Logf("Successfully fetched %s", url)
		}
		fmt.Println("=== Live URL test finished ===")
	})
}

