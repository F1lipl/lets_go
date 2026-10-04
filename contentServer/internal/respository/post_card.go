package respository

import (
	"context"
	"errors"
	"fmt"

	"contentserver/internal/domain"
	"contentserver/internal/model"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var (
	ErrPostCardSuperseded    = errors.New("post card publication superseded")
	ErrPostCardSourceMissing = errors.New("post card source missing")
	ErrPostCardSourceInvalid = errors.New("post card source invalid")
)

type PostCardPublication struct {
	PostID      domain.PostID
	AuthorID    domain.UserID
	RevisionID  domain.RevisionID
	PostVersion uint64
}

// PostCardRepository applies a publication using the transaction owned by the
// task repository. The card write and delivery completion therefore commit or
// roll back together.
type PostCardRepository struct{}

func NewPostCardRepository() *PostCardRepository { return &PostCardRepository{} }

func (*PostCardRepository) ApplyPublication(ctx context.Context, tx sqlx.Session, publication PostCardPublication) error {
	if tx == nil || publication.PostID.IsZero() || publication.AuthorID.IsZero() ||
		publication.RevisionID.IsZero() || publication.PostVersion == 0 {
		return ErrPostCardSourceInvalid
	}
	conn := sqlx.NewSqlConnFromSession(tx)
	postID := publication.PostID.String()
	revisionID := publication.RevisionID.String()

	// This is the same first lock used by publication and deletion, so neither
	// can change the current revision while the projection is being written.
	post, err := model.NewPostModel(conn).FindOneForUpdate(ctx, postID)
	if errors.Is(err, model.ErrNotFound) {
		return fmt.Errorf("%w: post %s", ErrPostCardSourceMissing, postID)
	}
	if err != nil {
		return err
	}
	if post.AuthorId != publication.AuthorID.String() || post.PostVersion < publication.PostVersion {
		return fmt.Errorf("%w: publication does not match post %s", ErrPostCardSourceInvalid, postID)
	}
	if post.LifecycleStatus == uint64(domain.LifecycleDeleted) || post.DeletedAt.Valid ||
		!post.PublishedRevisionId.Valid || post.PublishedRevisionId.String != revisionID {
		return ErrPostCardSuperseded
	}
	if post.LifecycleStatus != uint64(domain.LifecyclePublished) {
		return fmt.Errorf("%w: post %s is not published", ErrPostCardSourceInvalid, postID)
	}

	revision, err := model.NewPostRevisionModel(conn).FindCardSnapshot(ctx, revisionID)
	if errors.Is(err, model.ErrNotFound) {
		return fmt.Errorf("%w: revision %s", ErrPostCardSourceMissing, revisionID)
	}
	if err != nil {
		return err
	}
	if revision.PostID != postID || !revision.CoverAssetID.Valid || revision.CoverAssetID.String == "" {
		return fmt.Errorf("%w: revision %s has invalid card fields", ErrPostCardSourceInvalid, revisionID)
	}

	cards := model.NewPostCardProjectionModel(conn)
	current, err := cards.FindOne(ctx, postID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return err
	}
	if current != nil && current.SourcePostVersion >= post.PostVersion {
		return nil
	}
	card := &model.PostCardProjection{
		PostId: postID, RevisionId: revision.RevisionID,
		AuthorId: post.AuthorId, Title: revision.Title, Summary: revision.Summary,
		CoverAssetId: revision.CoverAssetID, Visibility: post.Visibility,
		AvailabilityStatus: post.AvailabilityStatus, PublishedAt: revision.PublishedAt,
		SourcePostVersion: post.PostVersion,
	}
	if current == nil {
		_, err = cards.Insert(ctx, card)
	} else {
		err = cards.Update(ctx, card)
	}
	return err
}
