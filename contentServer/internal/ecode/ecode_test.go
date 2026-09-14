package ecode

import (
	"errors"
	"net/http"
	"testing"
)

func TestCodesAreUniqueAndHaveMessages(t *testing.T) {
	codes := []Code{
		Success,
		InvalidRequest, InvalidCursor, InvalidPageSize, InvalidPostID, InvalidRequestID,
		PostNotFound, PostAlreadyDeleted, PostOperationNotAllowed, PostNotPublished, PostNotVisible,
		DraftNotFound, DraftVersionConflict, DraftContentInvalid,
		PublishNotAllowed, RevisionNotFound, RevisionCreateFailed,
		MediaAssetNotFound, MediaAssetNotReady, MediaTypeUnsupported, MediaSizeExceeded, MediaAssetInUse,
		TagNotFound, TagNameInvalid,
		RouteDraftNotFound, RouteVersionConflict, RouteSnapshotFailed,
		PostVersionConflict, RequestIdentityInvalid,
		InternalError, DatabaseError, CacheError, DependencyUnavailable,
	}

	seen := make(map[Code]struct{}, len(codes))
	for _, code := range codes {
		if _, exists := seen[code]; exists {
			t.Fatalf("duplicate code: %d", code)
		}
		seen[code] = struct{}{}

		if _, exists := messages[code]; !exists {
			t.Fatalf("missing message for code: %d", code)
		}
	}
}

func TestUnknownCodeFallsBackToInternalError(t *testing.T) {
	unknown := Code(123456789)
	if got, want := unknown.Message(), InternalError.Message(); got != want {
		t.Fatalf("Message() = %q, want %q", got, want)
	}
	if got := FromError(New(unknown)); got != InternalError {
		t.Fatalf("FromError() = %d, want %d", got, InternalError)
	}
}

func TestWrappedErrorPreservesCause(t *testing.T) {
	cause := errors.New("database timeout")
	err := Wrap(DatabaseError, cause)

	if got := FromError(err); got != DatabaseError {
		t.Fatalf("FromError() = %d, want %d", got, DatabaseError)
	}
	if !errors.Is(err, cause) {
		t.Fatal("wrapped error does not preserve cause")
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		code Code
		want int
	}{
		{Success, http.StatusOK},
		{InvalidRequest, http.StatusBadRequest},
		{PostNotFound, http.StatusNotFound},
		{DraftVersionConflict, http.StatusConflict},
		{DependencyUnavailable, http.StatusServiceUnavailable},
		{InternalError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		if got := HTTPStatus(tt.code); got != tt.want {
			t.Fatalf("HTTPStatus(%d) = %d, want %d", tt.code, got, tt.want)
		}
	}
}
