package respository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"contentserver/internal/domain"
	"contentserver/internal/model"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ domain.PostRepositoryInterface = (*PostRepository)(nil)

// PublishPost owns the transaction. apply must only perform in-memory domain work.
// No result is exposed until commit succeeds.
func (r *PostRepository) PublishPost(
	ctx context.Context, conn sqlx.SqlConn, postID domain.PostID,
	apply func(*domain.Post, *domain.PostDraft) (*domain.PostRevision, error),
) (*domain.PublicationResult, error) {
	if postID.IsZero() {
		return nil, domain.ErrInvalidPostID
	}
	if apply == nil {
		return nil, fmt.Errorf("publication callback is nil")
	}
	eventID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	var result *domain.PublicationResult
	err = conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		txConn := sqlx.NewSqlConnFromSession(session)
		postModel := model.NewPostModel(txConn)
		row, err := postModel.FindOneForUpdate(ctx, postID.String())
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return domain.ErrPostNotFound
			}
			return err
		}
		draftRow, err := model.NewPostDraftModel(txConn).FindOneForUpdate(ctx, postID.String())
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return domain.ErrDraftNotFound
			}
			return err
		}
		post, err := restorePost(row)
		if err != nil {
			return err
		}
		draft, err := restoreDraft(draftRow)
		if err != nil {
			return err
		}
		revision, err := apply(post, draft)
		if err != nil {
			return err
		}
		// Check callback output before performing any writes.
		if revision == nil || post.PublishedRevisionID == nil ||
			revision.PostId != postID || post.PostID != postID ||
			post.AuthorID.String() != row.AuthorId ||
			revision.RevisionId.IsZero() || *post.PublishedRevisionID != revision.RevisionId ||
			revision.RevisionNumber != row.RevisionSequence+1 ||
			post.RevisionSequence != revision.RevisionNumber ||
			post.Version != row.PostVersion+1 ||
			post.LifecycleStatus != domain.LifecyclePublished ||
			post.FirstPublishedAt == nil || post.LastPublishedAt == nil ||
			revision.SourceDraftVersion != draftRow.DraftVersion {
			return fmt.Errorf("invalid publication callback result")
		}
		if err := checkPublicationAssets(ctx, txConn, postID.String(), row.AuthorId, revision); err != nil {
			return err
		}
		if _, err := model.NewPostRevisionModel(txConn).Insert(ctx, revisionRow(revision)); err != nil {
			return err
		}
		row.PublishedRevisionId = sql.NullString{String: revision.RevisionId.String(), Valid: true}
		row.RevisionSequence = post.RevisionSequence
		row.PostVersion = post.Version
		row.LifecycleStatus = uint64(post.LifecycleStatus)
		row.FirstPublishedAt = sql.NullTime{Time: *post.FirstPublishedAt, Valid: true}
		row.LastPublishedAt = sql.NullTime{Time: *post.LastPublishedAt, Valid: true}
		if err := postModel.Update(ctx, row); err != nil {
			return err
		}

		// Retain immutable resource references; downstream projections use revision data,
		// never the mutable draft. Cover is retained even if the draft projection is empty.
		if _, err := txConn.ExecCtx(ctx,
			"INSERT INTO content_asset_ref (owner_type,owner_id,usage_type,block_id,asset_id,sort_order,source_version) SELECT 2,?,usage_type,block_id,asset_id,sort_order,? FROM content_asset_ref WHERE owner_type=1 AND owner_id=? AND usage_type=2",
			revision.RevisionId.String(), post.Version, postID.String()); err != nil {
			return err
		}
		if revision.Cover != nil {
			if _, err := txConn.ExecCtx(ctx,
				"INSERT INTO content_asset_ref (owner_type,owner_id,usage_type,block_id,asset_id,sort_order,source_version) VALUES (2,?,1,'',?,0,?)",
				revision.RevisionId.String(), revision.Cover.AssetID.String(), post.Version); err != nil {
				return err
			}
		}
		payload, err := json.Marshal(struct {
			SchemaVersion      int      `json:"schemaVersion"`
			PostID             string   `json:"postId"`
			AuthorID           string   `json:"authorId"`
			RevisionID         string   `json:"revisionId"`
			RevisionNumber     uint64   `json:"revisionNumber"`
			SourceDraftVersion uint64   `json:"sourceDraftVersion"`
			PostVersion        uint64   `json:"postVersion"`
			TagNames           []string `json:"tagNames"`
		}{1, postID.String(), post.AuthorID.String(), revision.RevisionId.String(), revision.RevisionNumber, revision.SourceDraftVersion, post.Version, revision.TagNames})
		if err != nil {
			return err
		}
		if _, err := txConn.ExecCtx(ctx,
			"INSERT INTO outbox_event (event_id,aggregate_type,aggregate_id,event_type,payload_json,occurred_at) VALUES (?,'Post',?,'PostPublished',?,?)",
			eventID.String(), postID.String(), string(payload), revision.PublishedAt); err != nil {
			return err
		}
		result = &domain.PublicationResult{PostID: postID, RevisionID: revision.RevisionId, RevisionNumber: revision.RevisionNumber, PostVersion: post.Version, PublishedAt: revision.PublishedAt}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Draft resource references must be maintained in the same transaction as its version.
// Shared row locks keep resources available until the revision references are committed.
func checkPublicationAssets(ctx context.Context, conn sqlx.SqlConn, postID, authorID string, revision *domain.PostRevision) error {
	var refs []struct {
		AssetID       string `db:"asset_id"`
		SourceVersion uint64 `db:"source_version"`
	}
	if err := conn.QueryRowsCtx(ctx, &refs, "SELECT asset_id,source_version FROM content_asset_ref WHERE owner_type=1 AND owner_id=? AND usage_type=2", postID); err != nil {
		return err
	}
	// Do not silently publish incomplete image references when draft persistence
	// has not maintained its projection. No document parsing is needed here.
	if uint64(len(refs)) != revision.ImageCount {
		return domain.ErrDraftContentInvalid
	}
	ids := make([]string, 0, len(refs)+1)
	for _, ref := range refs {
		if ref.SourceVersion != revision.SourceDraftVersion {
			return domain.ErrDraftContentInvalid
		}
		ids = append(ids, ref.AssetID)
	}
	if revision.Cover != nil {
		ids = append(ids, revision.Cover.AssetID.String())
	}
	unique := make(map[string]bool)
	for _, id := range ids {
		unique[id] = true
	}
	ids = ids[:0]
	for id := range unique {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Strings(ids)
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	var assets []struct {
		AssetID   string       `db:"asset_id"`
		OwnerID   string       `db:"owner_id"`
		Status    uint64       `db:"status"`
		DeletedAt sql.NullTime `db:"deleted_at"`
	}
	query := "SELECT asset_id,owner_id,status,deleted_at FROM media_asset WHERE asset_id IN (" + strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",") + ") ORDER BY asset_id FOR SHARE"
	if err := conn.QueryRowsCtx(ctx, &assets, query, args...); err != nil {
		return err
	}
	if len(assets) != len(ids) {
		return domain.ErrMediaAssetNotReady
	}
	for _, asset := range assets {
		if asset.OwnerID != authorID || asset.Status != 3 || asset.DeletedAt.Valid {
			return domain.ErrMediaAssetNotReady
		}
	}
	return nil
}
