package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"contentserver/internal/ecode"
	"contentserver/internal/httpresponse"
	"contentserver/internal/types"
)

func TestWriteBusinessResponseRejectsNilData(t *testing.T) {
	recorder := httptest.NewRecorder()
	var data *types.CreatePostData

	writeBusinessResponse(context.Background(), recorder, data, nil)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	var body httpresponse.Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.ErrorCode != ecode.InternalError.Int() {
		t.Fatalf("errorCode = %d, want %d", body.ErrorCode, ecode.InternalError)
	}
}

func TestWriteInvalidRequestMapsBatchPostIDs(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := types.BatchGetPostCardsRequest{}
	err := req.Validate()

	writeInvalidRequest(context.Background(), recorder, err)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	var body httpresponse.Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.ErrorCode != ecode.InvalidBatchPostIDs.Int() {
		t.Fatalf("errorCode = %d, want %d", body.ErrorCode, ecode.InvalidBatchPostIDs)
	}
}
