package httpresponse

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"contentserver/internal/ecode"
)

func TestErrorHandlerUsesApplicationCode(t *testing.T) {
	status, body := ErrorHandler(
		context.Background(),
		ecode.Wrap(ecode.PostNotFound, errors.New("row missing")),
	)

	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
	response, ok := body.(ErrorBody)
	if !ok {
		t.Fatalf("body type = %T, want ErrorBody", body)
	}
	if response.ErrorCode != ecode.PostNotFound.Int() {
		t.Fatalf("errorCode = %d, want %d", response.ErrorCode, ecode.PostNotFound)
	}
}

func TestSuccessHandlerFillsCommonFields(t *testing.T) {
	body := &stubResult{}
	result := SuccessHandler(context.Background(), body)

	if result != body {
		t.Fatal("SuccessHandler changed the response object")
	}
	if body.errorCode != ecode.Success.Int() {
		t.Fatalf("errorCode = %d, want %d", body.errorCode, ecode.Success)
	}
	if body.message != ecode.Success.Message() {
		t.Fatalf("message = %q, want %q", body.message, ecode.Success.Message())
	}
}

func TestErrorHandlerHidesUnknownError(t *testing.T) {
	status, body := ErrorHandler(context.Background(), errors.New("query failed"))
	response := body.(ErrorBody)

	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", status, http.StatusInternalServerError)
	}
	if response.ErrorCode != ecode.InternalError.Int() {
		t.Fatalf("errorCode = %d, want %d", response.ErrorCode, ecode.InternalError)
	}
	if response.Message != ecode.InternalError.Message() {
		t.Fatalf("message = %q, want %q", response.Message, ecode.InternalError.Message())
	}
}

type stubResult struct {
	errorCode int
	message   string
}

func (s *stubResult) SetResult(errorCode int, message string) {
	s.errorCode = errorCode
	s.message = message
}
