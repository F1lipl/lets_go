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
	BlockID       string    `json:"blockId"`
	ParentBlockID string    `json:"parentBlockId,omitempty"`
	BlockType     BlockType `json:"blockType"`
	SortOrder     int64     `json:"sortOrder"`

	Title    string    `json:"title,omitempty"`
	Text     string    `json:"text,omitempty"`
	AssetIDs []AssetID `json:"assetIds,omitempty"`
}

type DraftAssetRef struct {
	BlockID   string
	AssetID   AssetID
	SortOrder uint64
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
	AssetRefs             []DraftAssetRef // Transient index entries, derived when accepting new content.

	Version uint64

	CreatedAt time.Time
	UpdatedAt time.Time
}

func CreateNewPostDraft(id PostID, content *DraftContent, now time.Time) (*PostDraft, error) {
	if id.IsZero() {
		return nil, ErrInvalidPostID
	}
	if content == nil {
		return nil, ErrDraftContentInvalid
	}
	return &PostDraft{
		PostID:                id,
		Title:                 content.Title,
		Summary:               content.Summary,
		Cover:                 cloneCover(content.Cover),
		Document:              content.Document,
		DocumentSchemaVersion: content.DocumentSchemaVersion,
		PlainText:             content.PlainText,
		BlockCount:            content.BlockCount,
		ImageCount:            content.ImageCount,
		TagNames:              append(make([]string, 0, len(content.TagNames)), content.TagNames...),
		AssetRefs:             append([]DraftAssetRef(nil), content.AssetRefs...),
		Version:               1,
		CreatedAt:             now,
		UpdatedAt:             now,
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
