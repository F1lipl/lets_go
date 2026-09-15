package respository

import (
	"errors"
	"testing"

	"contentserver/internal/domain"
	"contentserver/internal/model"

	"github.com/google/uuid"
)

func TestClassifyDeleteFailure(t *testing.T) {
	authorID, err := domain.ParseUserID(uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	otherAuthorID := uuid.NewString()

	tests := []struct {
		name    string
		current *model.Post
		want    error
	}{
		{name: "missing", current: nil, want: domain.ErrPostNotFound},
		{
			name: "different author",
			current: &model.Post{
				AuthorId:        otherAuthorID,
				LifecycleStatus: uint64(domain.LifecyclePublished),
				PostVersion:     7,
			},
			want: domain.ErrPostOperationNotAllowed,
		},
		{
			name: "already deleted",
			current: &model.Post{
				AuthorId:        authorID.String(),
				LifecycleStatus: uint64(domain.LifecycleDeleted),
				PostVersion:     8,
			},
			want: domain.ErrPostAlreadyDeleted,
		},
		{
			name: "stale version",
			current: &model.Post{
				AuthorId:        authorID.String(),
				LifecycleStatus: uint64(domain.LifecyclePublished),
				PostVersion:     8,
			},
			want: domain.ErrPostVersionConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyDeleteFailure(tt.current, authorID, 7)
			if !errors.Is(got, tt.want) {
				t.Fatalf("classifyDeleteFailure() = %v, want %v", got, tt.want)
			}
		})
	}
}
