// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"time"

	"contentserver/internal/domain"
	"contentserver/internal/ecode"
	"contentserver/internal/identity"
	"contentserver/internal/respository"
	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreatePostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreatePostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePostLogic {
	return &CreatePostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreatePostLogic) CreatePost(req *types.CreatePostRequest) (*types.CreatePostData, error) {
	if req == nil {
		return nil, ecode.New(ecode.InvalidRequest)
	}
	currentUser, err := identity.FromContext(l.ctx)
	if err != nil {
		return nil, ecode.Wrap(ecode.RequestIdentityInvalid, err)
	}
	authorID, err := domain.ParseUserID(currentUser.UserID)
	if err != nil {
		return nil, err
	}
	visibility, err := domain.ParseVisibility(req.Visibility)
	if err != nil {
		return nil, err
	}
	content, err := draftContentFromRequest(req.Title, req.Summary, req.Cover, req.Document, req.TagNames)
	if err != nil {
		return nil, err
	}
	post, err := domain.NewPost(authorID, visibility)
	if err != nil {
		return nil, err
	}
	draft, err := domain.CreateNewPostDraft(post.PostID, content, post.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := respository.NewPostRepository().CreatePost(l.ctx, l.svcCtx.DB, post, draft); err != nil {
		return nil, err
	}
	logCommittedPostOperation(l.ctx, "create_post", post.PostID.String(), authorID.String(),
		logx.Field("postVersion", post.Version),
		logx.Field("draftVersion", draft.Version))
	return &types.CreatePostData{
		PostId: post.PostID.String(), Status: "draft", PostVersion: post.Version,
		DraftVersion: draft.Version, CreatedAt: post.CreatedAt.Format(time.RFC3339Nano),
	}, nil
}
