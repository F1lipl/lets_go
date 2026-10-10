package respository

import (
	"context"
	"errors"
	"testing"

	"contentserver/internal/domain"

	"github.com/google/uuid"
)

func TestDeletePostRejectsInvalidInputsBeforeTransaction(t *testing.T) {
	repo := NewPostRepository()
	if _, err := repo.DeletePost(context.Background(), nil, domain.PostID{}, func(*domain.Post) error { return nil }); !errors.Is(err, domain.ErrInvalidPostID) {
		t.Fatalf("zero post ID = %v", err)
	}
	postID, err := domain.ParsePostID(uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.DeletePost(context.Background(), nil, postID, nil); err == nil {
		t.Fatal("nil delete callback accepted")
	}
}
