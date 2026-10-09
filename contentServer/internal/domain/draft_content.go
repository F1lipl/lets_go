package domain

import (
	"encoding/json"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxDraftDocumentBytes = 1 << 20
	maxDraftBlocks        = 256
	maxDraftImages        = 100
)

// DraftContent is a validated full replacement for the editable draft.
// Document stays serialized; its derived fields are calculated only on write.
type DraftContent struct {
	Title                 string
	Summary               string
	Cover                 *Cover
	Document              string
	DocumentSchemaVersion uint64
	PlainText             string
	BlockCount            uint64
	ImageCount            uint64
	TagNames              []string
	AssetRefs             []DraftAssetRef
}

func NewDraftContent(title, summary string, cover *Cover, document string, tagNames []string) (*DraftContent, error) {
	if !utf8.ValidString(title) || !utf8.ValidString(summary) ||
		utf8.RuneCountInString(title) > 120 || utf8.RuneCountInString(summary) > 500 ||
		strings.IndexFunc(title, unicode.IsControl) >= 0 {
		return nil, ErrDraftContentInvalid
	}
	if cover != nil {
		if cover.AssetID.IsZero() || !validFocus(cover.FocusX) || !validFocus(cover.FocusY) ||
			len(cover.CropStyle) > 32 || !printableASCII(cover.CropStyle) {
			return nil, ErrDraftContentInvalid
		}
	}
	if len(document) == 0 || len(document) > maxDraftDocumentBytes || !utf8.ValidString(document) {
		return nil, ErrDraftContentInvalid
	}
	var parsed PostDocument
	if err := json.Unmarshal([]byte(document), &parsed); err != nil || parsed.SchemaVersion != 1 || len(parsed.Blocks) > maxDraftBlocks {
		return nil, ErrDraftContentInvalid
	}
	preparedTags, err := PrepareTagNames(tagNames)
	if err != nil {
		return nil, err
	}
	content := &DraftContent{
		Title: title, Summary: summary, Cover: cloneCover(cover), Document: document,
		DocumentSchemaVersion: parsed.SchemaVersion, BlockCount: uint64(len(parsed.Blocks)),
		TagNames: make([]string, 0, len(preparedTags)),
	}
	for _, tag := range preparedTags {
		content.TagNames = append(content.TagNames, tag.Display)
	}
	blockByID := make(map[string]ContentBlock, len(parsed.Blocks))
	var plain strings.Builder
	for _, block := range parsed.Blocks {
		if block.BlockID == "" || len(block.BlockID) > 64 || !printableASCII(block.BlockID) ||
			len(block.ParentBlockID) > 64 || !printableASCII(block.ParentBlockID) ||
			!block.BlockType.Valid() || block.SortOrder < 0 {
			return nil, ErrDraftContentInvalid
		}
		if _, duplicate := blockByID[block.BlockID]; duplicate {
			return nil, ErrDraftContentInvalid
		}
		blockByID[block.BlockID] = block
		if block.Title != "" {
			plain.WriteString(block.Title)
			plain.WriteByte('\n')
		}
		if block.Text != "" {
			plain.WriteString(block.Text)
			plain.WriteByte('\n')
		}
		for i, assetID := range block.AssetIDs {
			if assetID.IsZero() || len(content.AssetRefs) >= maxDraftImages {
				return nil, ErrDraftContentInvalid
			}
			content.AssetRefs = append(content.AssetRefs, DraftAssetRef{
				BlockID: block.BlockID, AssetID: assetID, SortOrder: uint64(i),
			})
		}
	}
	for _, block := range parsed.Blocks {
		seen := make(map[string]bool)
		for parentID := block.ParentBlockID; parentID != ""; {
			if seen[parentID] || parentID == block.BlockID {
				return nil, ErrDraftContentInvalid
			}
			seen[parentID] = true
			parent, exists := blockByID[parentID]
			if !exists {
				return nil, ErrDraftContentInvalid
			}
			parentID = parent.ParentBlockID
		}
	}
	content.PlainText = plain.String()
	content.ImageCount = uint64(len(content.AssetRefs))
	return content, nil
}

func cloneCover(cover *Cover) *Cover {
	if cover == nil {
		return nil
	}
	copy := *cover
	return &copy
}

func validFocus(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

func printableASCII(value string) bool {
	for _, r := range value {
		if r < 32 || r > 126 {
			return false
		}
	}
	return true
}
