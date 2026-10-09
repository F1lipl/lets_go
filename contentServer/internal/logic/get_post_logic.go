// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"contentserver/internal/domain"
	"contentserver/internal/ecode"
	"contentserver/internal/respository"
	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type GetPostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPostLogic {
	return &GetPostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPostLogic) GetPost(req *types.GetPostRequest) (data *types.GetPostData, err error) {
	if req == nil {
		return nil, ecode.New(ecode.InvalidRequest)
	}
	postID, err := domain.ParsePostID(req.PostId)
	if err != nil {
		return nil, err
	}
	repo := respository.NewPostRepository()
	return readPublishedPost(l.ctx, repo, l.svcCtx.DB, postID, mediaLookupTimeout(l.svcCtx.Config.PostMediaLookupTimeoutMs))
}

const defaultMediaLookupTimeout = 300 * time.Millisecond

func mediaLookupTimeout(milliseconds int64) time.Duration {
	if milliseconds <= 0 {
		return defaultMediaLookupTimeout
	}
	return time.Duration(milliseconds) * time.Millisecond
}

func readPublishedPost(ctx context.Context, queries respository.PostReadQueries, conn sqlx.SqlConn, postID domain.PostID, mediaTimeout time.Duration) (*types.GetPostData, error) {
	post, err := queries.FindPost(ctx, conn, postID)
	if err != nil {
		return nil, err
	}
	revisionID, err := post.CurrentPublicRevision()
	if err != nil {
		return nil, err
	}
	revision, err := queries.FindRevision(ctx, conn, revisionID)
	if err != nil {
		return nil, err
	}
	if !revision.BelongsTo(postID) || revision.RevisionId != revisionID {
		return nil, fmt.Errorf("revision %s does not belong to post %s", revisionID, postID)
	}
	document, err := decodePublishedDocument(revision)
	if err != nil {
		return nil, err
	}
	assetIDs, err := referencedAssetIDs(revision.Cover, document)
	if err != nil {
		return nil, err
	}
	var assets []domain.PublishedAsset
	mediaUnavailable := false
	if len(assetIDs) > 0 {
		mediaCtx, cancel := context.WithTimeout(ctx, mediaTimeout)
		assets, err = queries.FindReadyAssets(mediaCtx, conn, assetIDs)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			logx.WithContext(ctx).Errorf("post %s media metadata unavailable: %v", postID, err)
			mediaUnavailable = true
			assets = nil
		}
	}
	tags, err := queries.FindTags(ctx, conn, revisionID)
	if err != nil {
		return nil, err
	}
	counts, err := queries.FindCounts(ctx, conn, postID)
	if err != nil {
		return nil, err
	}
	return publishedPostData(post, revision, document, assetIDs, assets, tags, counts, mediaUnavailable)
}

func referencedAssetIDs(cover *domain.Cover, document types.PostDocument) ([]domain.AssetID, error) {
	ids := make([]domain.AssetID, 0)
	seen := make(map[domain.AssetID]struct{})
	add := func(id domain.AssetID) {
		if id.IsZero() {
			return
		}
		if _, exists := seen[id]; exists {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if cover != nil {
		add(cover.AssetID)
	}
	for _, block := range document.Blocks {
		for _, rawID := range block.AssetIds {
			id, err := domain.ParseAssetID(rawID)
			if err != nil {
				return nil, fmt.Errorf("invalid published asset reference: %w", err)
			}
			add(id)
		}
	}
	return ids, nil
}

func decodePublishedDocument(revision *domain.PostRevision) (types.PostDocument, error) {
	var document types.PostDocument
	if revision == nil {
		return document, fmt.Errorf("published revision is missing")
	}
	if err := json.Unmarshal([]byte(revision.Document), &document); err != nil {
		return document, fmt.Errorf("decode published document %s: %w", revision.RevisionId, err)
	}
	if document.SchemaVersion == 0 || uint64(document.SchemaVersion) != revision.DocumentSchemaVersion {
		return document, fmt.Errorf("published document schema mismatch for revision %s", revision.RevisionId)
	}
	if document.Blocks == nil {
		document.Blocks = []types.ContentBlock{}
	}
	return document, nil
}

func publishedPostData(post *domain.Post, revision *domain.PostRevision, document types.PostDocument, assetIDs []domain.AssetID, assets []domain.PublishedAsset, tags []domain.PublishedTag, counts *domain.PostCounts, mediaUnavailable bool) (*types.GetPostData, error) {
	if post == nil || revision == nil {
		return nil, fmt.Errorf("published post data is incomplete")
	}
	byID := make(map[domain.AssetID]domain.PublishedAsset, len(assets))
	for _, asset := range assets {
		byID[asset.ID] = asset
	}
	media := make([]types.MediaAssetView, 0, len(assetIDs))
	cover := types.CoverView{}
	if revision.Cover != nil {
		cover.AssetId = revision.Cover.AssetID.String()
		cover.FocusX = revision.Cover.FocusX
		cover.FocusY = revision.Cover.FocusY
		cover.CropStyle = revision.Cover.CropStyle
	}
	for _, id := range assetIDs {
		asset, exists := byID[id]
		if !exists {
			status := "missing"
			if mediaUnavailable {
				status = "unavailable"
			}
			media = append(media, types.MediaAssetView{AssetId: id.String(), Status: status})
			continue
		}
		media = append(media, types.MediaAssetView{
			AssetId: asset.ID.String(), MimeType: asset.MimeType, Status: "ready",
			Width: asset.Width, Height: asset.Height,
		})
		if revision.Cover != nil && asset.ID == revision.Cover.AssetID {
			cover.Width, cover.Height = asset.Width, asset.Height
		}
	}
	tagViews := make([]types.PostTag, 0, len(tags))
	for _, tag := range tags {
		tagViews = append(tagViews, types.PostTag{Id: tag.ID, Name: tag.Name})
	}
	if counts == nil {
		counts = &domain.PostCounts{}
	}
	return &types.GetPostData{
		PostId: post.PostID.String(), AuthorId: post.AuthorID.String(),
		Status: "published", Visibility: "public", RevisionId: revision.RevisionId.String(),
		RevisionNumber: revision.RevisionNumber, Title: revision.Title, Summary: revision.Summary,
		HasCover: revision.Cover != nil, Cover: cover,
		Document: document, MediaAssets: media, Tags: tagViews,
		LikeCount: counts.Likes, FavoriteCount: counts.Favorites,
		CommentCount: counts.Comments, PostVersion: post.Version,
		PublishedAt: revision.PublishedAt.Format(time.RFC3339Nano),
	}, nil
}
