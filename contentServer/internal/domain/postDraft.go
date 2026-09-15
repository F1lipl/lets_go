package domain

import "time"

type DraftCover struct {
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
	Cover   *DraftCover

	Document PostDocument
	TagNames []string

	Version uint64

	CreatedAt           time.Time
	UpdatedAt           time.Time
	PostDraftRepository postDraftRepository
}

func newPostDraft(id PostID, title string, summary string, cover *DraftCover, document PostDocument, tagName []string) (*PostDraft, error) {
	if id.IsZero() {
		return nil, ErrInvalidPostID
	}
	now := time.Now()
	return &PostDraft{
		PostID:    id,
		Title:     title,
		Summary:   summary,
		Cover:     cover,
		TagNames:  tagName,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
func (post *PostDraft) SaveDraft() error {
	return nil
}
