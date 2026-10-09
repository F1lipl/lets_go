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

type SavePostDraftLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSavePostDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SavePostDraftLogic {
	return &SavePostDraftLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SavePostDraftLogic) SavePostDraft(req *types.SavePostDraftRequest) (*types.SavePostDraftData, error) {
	if req == nil {
		return nil, ecode.New(ecode.InvalidRequest)
	}
	currentUser, err := identity.FromContext(l.ctx)
	if err != nil {
		return nil, ecode.Wrap(ecode.RequestIdentityInvalid, err)
	}
	actorID, err := domain.ParseUserID(currentUser.UserID)
	if err != nil {
		return nil, err
	}
	postID, err := domain.ParsePostID(req.PostId)
	if err != nil {
		return nil, err
	}
	if req.ExpectedDraftVersion == 0 {
		return nil, domain.ErrInvalidVersion
	}
	content, err := draftContentFromRequest(req.Title, req.Summary, req.Cover, req.Document, req.TagNames)
	if err != nil {
		return nil, err
	}
	saved, err := respository.NewPostRepository().SavePostDraft(l.ctx, l.svcCtx.DB, postID,
		func(post *domain.Post, currentVersion uint64) (*domain.PostDraft, error) {
			return post.ReplaceDraft(actorID, currentVersion, req.ExpectedDraftVersion, content,
				time.Now().UTC().Truncate(time.Millisecond))
		})
	if err != nil {
		return nil, err
	}
	return &types.SavePostDraftData{
		PostId: saved.PostID.String(), DraftVersion: saved.Version,
		UpdatedAt: saved.UpdatedAt.Format(time.RFC3339Nano),
	}, nil
}
