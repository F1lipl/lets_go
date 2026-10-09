package respository

import (
	"context"
	"errors"

	"contentserver/internal/domain"
	"contentserver/internal/model"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// PostReadQueries exposes only database reads. The caller decides which
// version may be shown and assembles the response.
type PostReadQueries interface {
	FindPost(context.Context, sqlx.SqlConn, domain.PostID) (*domain.Post, error)
	FindRevision(context.Context, sqlx.SqlConn, domain.RevisionID) (*domain.PostRevision, error)
	FindReadyAssets(context.Context, sqlx.SqlConn, []domain.AssetID) ([]domain.PublishedAsset, error)
	FindTags(context.Context, sqlx.SqlConn, domain.RevisionID) ([]domain.PublishedTag, error)
	FindCounts(context.Context, sqlx.SqlConn, domain.PostID) (*domain.PostCounts, error)
}

var _ PostReadQueries = (*PostRepository)(nil)

func (r *PostRepository) FindPost(ctx context.Context, conn sqlx.SqlConn, postID domain.PostID) (*domain.Post, error) {
	postRow, err := model.NewPostModel(conn).FindOne(ctx, postID.String())
	if errors.Is(err, model.ErrNotFound) {
		return nil, domain.ErrPostNotFound
	}
	if err != nil {
		return nil, err
	}
	return restorePost(postRow)
}

func (r *PostRepository) FindRevision(ctx context.Context, conn sqlx.SqlConn, revisionID domain.RevisionID) (*domain.PostRevision, error) {
	revisionRow, err := model.NewPostRevisionModel(conn).FindOne(ctx, revisionID.String())
	if errors.Is(err, model.ErrNotFound) {
		return nil, domain.ErrRevisionNotFound
	}
	if err != nil {
		return nil, err
	}
	return restoreRevision(revisionRow)
}

func restoreRevision(row *model.PostRevision) (*domain.PostRevision, error) {
	revisionID, err := domain.ParseRevisionID(row.RevisionId)
	if err != nil {
		return nil, err
	}
	postID, err := domain.ParsePostID(row.PostId)
	if err != nil {
		return nil, err
	}
	revision := &domain.PostRevision{
		RevisionId: revisionID, PostId: postID, RevisionNumber: row.RevisionNumber,
		SourceDraftVersion: row.SourceDraftVersion, DocumentSchemaVersion: row.DocumentSchemaVersion,
		PlainText: row.PlainText, BlockCount: row.BlockCount, ImageCount: row.ImageCount,
		Title: row.Title, Summary: row.Summary, Document: row.DocumentJson, PublishedAt: row.PublishedAt,
	}
	if row.CoverAssetId.Valid {
		assetID, err := domain.ParseAssetID(row.CoverAssetId.String)
		if err != nil {
			return nil, err
		}
		revision.Cover = &domain.Cover{
			AssetID: assetID, FocusX: row.CoverFocusX.Float64,
			FocusY: row.CoverFocusY.Float64, CropStyle: row.CoverCropStyle,
		}
	}
	return revision, nil
}

func (r *PostRepository) FindReadyAssets(ctx context.Context, conn sqlx.SqlConn, ids []domain.AssetID) ([]domain.PublishedAsset, error) {
	values := make([]string, len(ids))
	for i, id := range ids {
		values[i] = id.String()
	}
	rows, err := model.NewMediaAssetModel(conn).FindReadyByIDs(ctx, values)
	if err != nil {
		return nil, err
	}
	assets := make([]domain.PublishedAsset, 0, len(rows))
	for _, row := range rows {
		id, err := domain.ParseAssetID(row.AssetId)
		if err != nil {
			return nil, err
		}
		assets = append(assets, domain.PublishedAsset{
			ID: id, MimeType: row.MimeType,
			Width: row.Width.Int64, Height: row.Height.Int64,
		})
	}
	return assets, nil
}

func (r *PostRepository) FindTags(ctx context.Context, conn sqlx.SqlConn, revisionID domain.RevisionID) ([]domain.PublishedTag, error) {
	rows, err := model.NewPostRevisionTagModel(conn).FindByRevisionId(ctx, revisionID.String())
	if err != nil {
		return nil, err
	}
	tags := make([]domain.PublishedTag, 0, len(rows))
	for _, row := range rows {
		tags = append(tags, domain.PublishedTag{ID: row.TagId, Name: row.TagNameSnapshot})
	}
	return tags, nil
}

func (r *PostRepository) FindCounts(ctx context.Context, conn sqlx.SqlConn, postID domain.PostID) (*domain.PostCounts, error) {
	row, err := model.NewPostStatsModel(conn).FindOne(ctx, postID.String())
	if errors.Is(err, model.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.PostCounts{
		Likes: row.LikeCount, Favorites: row.FavoriteCount, Comments: row.CommentCount,
	}, nil
}
