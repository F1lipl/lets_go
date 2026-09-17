package domain

import (
	"strings"
	"time"
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

type PostDraft struct {
	PostID PostID

	Title   string
	Summary string
	Cover   *Cover

	Document              string
	DocumentSchemaVersion uint64
	PlainText             string
	BlockCount            uint64
	ImageCount            uint64
	TagNames              []string

	Version uint64

	CreatedAt time.Time
	UpdatedAt time.Time
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

// Stored document structure is validated when saving a new draft, not reparsed here.
func (draft *PostDraft) ValidateForPublish() error {
	if draft.Version == 0 {
		return ErrInvalidVersion
	}
	if draft.Cover == nil || draft.Cover.AssetID.IsZero() {
		return ErrDraftContentInvalid
	}
	if strings.TrimSpace(draft.Document) == "" || draft.DocumentSchemaVersion == 0 {
		return ErrDraftContentInvalid
	}
	return nil
}
