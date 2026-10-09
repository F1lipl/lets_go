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
	return &GetPostLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetPostLogic) GetPost(req *types.GetPostRequest) (*types.GetPostData, error) {
	if req == nil {
		return nil, ecode.New(ecode.InvalidRequest)
	}
	postID, err := domain.ParsePostID(req.PostId)
	if err != nil {
		return nil, err
	}
	return readPublishedPost(l.ctx, respository.NewPostRepository(), l.svcCtx.DB, postID,
		supplementLookupTimeout(l.svcCtx.Config.PostSupplementLookupTimeoutMs))
}

const defaultSupplementLookupTimeout = 200 * time.Millisecond

func supplementLookupTimeout(milliseconds int64) time.Duration {
	if milliseconds <= 0 {
		return defaultSupplementLookupTimeout
	}
	return time.Duration(milliseconds) * time.Millisecond
}

func readPublishedPost(ctx context.Context, queries respository.PostReadQueries, conn sqlx.SqlConn, postID domain.PostID, supplementTimeout time.Duration) (*types.GetPostData, error) {
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
	data, err := publishedPostData(post, revision)
	if err != nil {
		return nil, err
	}

	// These reads are optional. One shared deadline bounds the total extra
	// latency after the immutable text has been loaded.
	supplementCtx, cancel := context.WithTimeout(ctx, supplementTimeout)
	defer cancel()
	tags, tagErr := queries.FindTags(supplementCtx, conn, revisionID)
	if tagErr == nil {
		data.TagsStatus = "ready"
		for _, tag := range tags {
			data.Tags = append(data.Tags, types.PostTag{Id: tag.ID, Name: tag.Name})
		}
	} else {
		logx.WithContext(ctx).Errorf("post %s tags unavailable: %v", postID, tagErr)
	}
	if supplementCtx.Err() == nil {
		counts, countErr := queries.FindCounts(supplementCtx, conn, postID)
		if countErr == nil && counts != nil {
			data.StatsStatus = "ready"
			data.LikeCount, data.FavoriteCount, data.CommentCount = counts.Likes, counts.Favorites, counts.Comments
		} else if countErr != nil {
			logx.WithContext(ctx).Errorf("post %s counts unavailable: %v", postID, countErr)
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return data, nil
}

func publishedPostData(post *domain.Post, revision *domain.PostRevision) (*types.GetPostData, error) {
	if post == nil || revision == nil {
		return nil, fmt.Errorf("published post data is incomplete")
	}
	cover := types.CoverView{}
	if revision.Cover != nil {
		cover.AssetId = revision.Cover.AssetID.String()
		cover.FocusX = revision.Cover.FocusX
		cover.FocusY = revision.Cover.FocusY
		cover.CropStyle = revision.Cover.CropStyle
	}
	return &types.GetPostData{
		PostId: post.PostID.String(), AuthorId: post.AuthorID.String(),
		Status: "published", Visibility: "public", RevisionId: revision.RevisionId.String(),
		RevisionNumber: revision.RevisionNumber, Title: revision.Title, Summary: revision.Summary,
		HasCover: revision.Cover != nil, Cover: cover,
		Document: json.RawMessage(revision.Document), Tags: []types.PostTag{}, TagsStatus: "unavailable",
		StatsStatus: "unavailable", PostVersion: post.Version,
		PublishedAt: revision.PublishedAt.Format(time.RFC3339Nano),
	}, nil
}
