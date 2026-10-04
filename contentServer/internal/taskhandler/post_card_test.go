package taskhandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"contentserver/internal/event"
	"contentserver/internal/respository"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type stubPostCardProjector struct {
	called      int
	publication respository.PostCardPublication
	err         error
}

func (s *stubPostCardProjector) ApplyPublication(_ context.Context, _ sqlx.Session, publication respository.PostCardPublication) error {
	s.called++
	s.publication = publication
	return s.err
}

func TestPostCardHandlerDelegatesAndClassifiesResult(t *testing.T) {
	postID, authorID, revisionID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	payload, err := json.Marshal(postPublishedPayload{
		SchemaVersion: 1, PostID: postID, AuthorID: authorID,
		RevisionID: revisionID, PostVersion: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	task := &event.TaskContext{EventType: "PostPublished", SchemaVersion: 1, Payload: payload}
	// The stub never calls the session; the test checks only the handler boundary.
	var tx sqlx.Session = struct{ sqlx.Session }{}
	transient := errors.New("temporary write failure")
	for _, tc := range []struct {
		name      string
		repoError error
		want      error
		permanent bool
	}{
		{name: "success"},
		{name: "superseded", repoError: respository.ErrPostCardSuperseded, want: event.ErrSuperseded},
		{name: "invalid source", repoError: respository.ErrPostCardSourceInvalid, want: respository.ErrPostCardSourceInvalid, permanent: true},
		{name: "missing source", repoError: respository.ErrPostCardSourceMissing, want: respository.ErrPostCardSourceMissing, permanent: true},
		{name: "transient", repoError: transient, want: transient},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubPostCardProjector{err: tc.repoError}
			err := NewPostCardHandler(repo).Handle(context.Background(), tx, task)
			if !errors.Is(err, tc.want) || event.IsPermanent(err) != tc.permanent {
				t.Fatalf("Handle() = %v, want %v (permanent=%t)", err, tc.want, tc.permanent)
			}
			if repo.called != 1 || repo.publication.PostID.String() != postID ||
				repo.publication.AuthorID.String() != authorID || repo.publication.RevisionID.String() != revisionID ||
				repo.publication.PostVersion != 7 {
				t.Fatalf("repository call = %d, publication = %+v", repo.called, repo.publication)
			}
		})
	}
}

func TestPostCardHandlerRejectsMalformedEventBeforeRepository(t *testing.T) {
	repo := &stubPostCardProjector{}
	var tx sqlx.Session = struct{ sqlx.Session }{}
	task := &event.TaskContext{EventType: "PostPublished", SchemaVersion: 1, Payload: []byte(`{"schemaVersion":1,"postId":"not-an-id"}`)}
	err := NewPostCardHandler(repo).Handle(context.Background(), tx, task)
	if !event.IsPermanent(err) || repo.called != 0 {
		t.Fatalf("Handle() = %v, repository calls = %d", err, repo.called)
	}
}
