package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type countingReader struct {
	reader *strings.Reader
	read   int
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += n
	return n, err
}

func TestDraftWriteHandlersLimitRequestSize(t *testing.T) {
	for _, handler := range []http.HandlerFunc{CreatePostHandler(nil), SavePostDraftHandler(nil)} {
		body := fmt.Sprintf(`{"visibility":"public","document":{"schemaVersion":1,"blocks":[{"blockId":"b1","blockType":"paragraph","text":%q}]}}`, strings.Repeat("x", maxDraftWriteRequestBytes))
		reader := &countingReader{reader: strings.NewReader(body)}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/posts", reader)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler(response, req)
		if response.Code == http.StatusOK || reader.read > maxDraftWriteRequestBytes+1 {
			t.Fatalf("oversized request: status=%d, read=%d", response.Code, reader.read)
		}
	}
}
