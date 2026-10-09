package respository

import (
	"database/sql"
	"encoding/json"

	"contentserver/internal/domain"
	"contentserver/internal/model"
)

// draftRow only maps an already validated domain value to the storage shape.
func draftRow(draft *domain.PostDraft) (*model.PostDraft, error) {
	tagsJSON, err := json.Marshal(draft.TagNames)
	if err != nil {
		return nil, err
	}
	row := &model.PostDraft{
		PostId:                draft.PostID.String(),
		DraftVersion:          draft.Version,
		Title:                 draft.Title,
		Summary:               draft.Summary,
		DocumentSchemaVersion: draft.DocumentSchemaVersion,
		DocumentJson:          draft.Document,
		TagNamesJson:          string(tagsJSON),
		PlainText:             draft.PlainText,
		BlockCount:            draft.BlockCount,
		ImageCount:            draft.ImageCount,
	}
	if draft.Cover != nil {
		row.CoverAssetId = sql.NullString{String: draft.Cover.AssetID.String(), Valid: true}
		row.CoverFocusX = sql.NullFloat64{Float64: draft.Cover.FocusX, Valid: true}
		row.CoverFocusY = sql.NullFloat64{Float64: draft.Cover.FocusY, Valid: true}
		row.CoverCropStyle = draft.Cover.CropStyle
	}
	return row, nil
}
