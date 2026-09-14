package httpresponse

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"contentserver/internal/ecode"
)

func TestWriteSuccessBuildsEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	data := struct {
		PostID string `json:"postId"`
	}{PostID: "post-1"}

	WriteSuccess(context.Background(), recorder, data)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var body struct {
		ErrorCode int `json:"errorCode"`
		Data      struct {
			PostID string `json:"postId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.ErrorCode != ecode.Success.Int() || body.Data.PostID != data.PostID {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestWriteErrorMapsApplicationError(t *testing.T) {
	recorder := httptest.NewRecorder()

	WriteError(
		context.Background(),
		recorder,
		ecode.Wrap(ecode.PostNotFound, errors.New("row missing")),
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	var body Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.ErrorCode != ecode.PostNotFound.Int() {
		t.Fatalf("errorCode = %d, want %d", body.ErrorCode, ecode.PostNotFound)
	}
	if body.Data != nil {
		t.Fatalf("error response contains data: %#v", body.Data)
	}
}

func TestWriteErrorHidesUnknownError(t *testing.T) {
	recorder := httptest.NewRecorder()

	WriteError(context.Background(), recorder, errors.New("query failed"))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	var body Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.ErrorCode != ecode.InternalError.Int() {
		t.Fatalf("errorCode = %d, want %d", body.ErrorCode, ecode.InternalError)
	}
	if body.Message != ecode.InternalError.Message() {
		t.Fatalf("message = %q, want %q", body.Message, ecode.InternalError.Message())
	}
}

func TestStatusFor(t *testing.T) {
	tests := []struct {
		code ecode.Code
		want int
	}{
		{ecode.Success, http.StatusOK},
		{ecode.InvalidRequest, http.StatusBadRequest},
		{ecode.PostNotFound, http.StatusNotFound},
		{ecode.DraftVersionConflict, http.StatusConflict},
		{ecode.DependencyUnavailable, http.StatusServiceUnavailable},
		{ecode.InternalError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		if got := statusFor(tt.code); got != tt.want {
			t.Fatalf("statusFor(%d) = %d, want %d", tt.code, got, tt.want)
		}
	}
}
