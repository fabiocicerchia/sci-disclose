// Package fetch is the one HTTP client in this tool: a short timeout, a
// bounded read, and a User-Agent that names the tool and its version so an API
// operator can see who is calling.
package fetch

import (
	"io"
	"net/http"
	"time"

	"github.com/fabiocicerchia/sci-disclose/internal/coefficients"
)

// Get returns the body, the status code, and any transport error. The body is
// capped at 1 MiB: every endpoint this tool reads is a small JSON document or a
// metrics page, and an unbounded read is a memory bug waiting for a bad server.
func Get(endpoint string) ([]byte, int, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("User-Agent", "sci-disclose/"+coefficients.Version)
	response, err := client.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	return body, response.StatusCode, err
}
