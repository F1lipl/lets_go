package domain

import (
	"contentserver/internal/model"
	"context"
	"encoding/json"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type Cover struct {
	AssetID   AssetID
	FocusX    float64
	FocusY    float64
	CropStyle string
}

type PostDocument struct {
	SchemaVersion uint64         `json:"schemaVersion"`
	Blocks        []ContentBlock `json:"blocks"`
}

type BlockType string

const (
	BlockParagraph BlockType = "paragraph"
	BlockHeading   BlockType = "heading"
	BlockImage     BlockType = "image"
)

func (t BlockType) Valid() bool {
	switch t {
	case BlockParagraph, BlockHeading, BlockImage:
		return true
	default:
		return false
	}
}

type ContentBlock struct {
	BlockID       string    `json:"BlockID"`
	ParentBlockID string    `json:"ParentBlockID,omitempty"`
	BlockType     BlockType `json:"BlockType"`
	SortOrder     int64     `json:"SortOrder"`

	Title    string    `json:"Title,omitempty"`
	Text     string    `json:"Text,omitempty"`
	AssetIDs []AssetID `json:"AssetIDs,omitempty"`
}

type postDraftRepository interface {
	SaveDraft() error
	PublishDraft() error
	DeleteDraft() error
}

type PostDraft struct {
	PostID PostID

	Title   string
	Summary string
	Cover   *Cover

	//Document PostDocument
	Document string
	TagNames []string

	Version uint64

	CreatedAt           time.Time
	UpdatedAt           time.Time
	PostDraftRepository postDraftRepository
}

func CreateNewPostDraft(id PostID, title string, summary string, cover *Cover, document string, tagName []string) (*PostDraft, error) {
	if id.IsZero() {
		return nil, ErrInvalidPostID
	}
	now := time.Now()
	return &PostDraft{
		PostID:    id,
		Title:     title,
		Summary:   summary,
		Cover:     cover,
		Document:  document,
		TagNames:  tagName,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
func getDraft(ctx context.Context, conn sqlx.SqlConn, id PostID) (*model.PostDraft, error) {
	postDraftModel := model.NewPostDraftModel(conn)
	return postDraftModel.FindOne(ctx, id.String())
}

func GetPostDraft(id PostID, ctx context.Context, conn sqlx.SqlConn) (*PostDraft, error) {
	postDraftModel, err := getDraft(ctx, conn, id)
	if err != nil {
		return nil, err
	}

	postID, err := ParsePostID(postDraftModel.PostId)
	if err != nil {
		return nil, err
	}

	var cover *Cover
	if postDraftModel.CoverAssetId.Valid {
		assetID, err := ParseAssetID(postDraftModel.CoverAssetId.String)
		if err != nil {
			return nil, err
		}
		cover = &Cover{
			AssetID:   assetID,
			FocusX:    postDraftModel.CoverFocusX.Float64,
			FocusY:    postDraftModel.CoverFocusY.Float64,
			CropStyle: postDraftModel.CoverCropStyle,
		}
	}

	//var document PostDocument
	//if err := json.Unmarshal([]byte(postDraftModel.DocumentJson), &document); err != nil {
	//	return nil, err
	//}

	var tagNames []string
	if err := json.Unmarshal([]byte(postDraftModel.TagNamesJson), &tagNames); err != nil {
		return nil, err
	}

	return &PostDraft{
		PostID:    postID,
		Title:     postDraftModel.Title,
		Summary:   postDraftModel.Summary,
		Cover:     cover,
		Document:  postDraftModel.DocumentJson,
		TagNames:  tagNames,
		Version:   postDraftModel.DraftVersion,
		CreatedAt: postDraftModel.CreatedAt,
		UpdatedAt: postDraftModel.UpdatedAt,
	}, nil
}

func (post *PostDraft) SaveDraft() error {
	return nil
}
