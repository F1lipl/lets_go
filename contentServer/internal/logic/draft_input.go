package logic

import (
	"encoding/json"

	"contentserver/internal/domain"
	"contentserver/internal/types"
)

// The API types are converted only at the transport boundary. The domain
// validates the document and derives all fields persisted alongside it.
func draftContentFromRequest(title, summary string, cover types.CoverInput, document types.PostDocument, tagNames []string) (*domain.DraftContent, error) {
	var domainCover *domain.Cover
	if cover.AssetId != "" {
		assetID, err := domain.ParseAssetID(cover.AssetId)
		if err != nil {
			return nil, err
		}
		domainCover = &domain.Cover{AssetID: assetID, FocusX: cover.FocusX, FocusY: cover.FocusY, CropStyle: cover.CropStyle}
	} else if cover.FocusX != 0 || cover.FocusY != 0 || cover.CropStyle != "" {
		return nil, domain.ErrDraftContentInvalid
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, domain.ErrDraftContentInvalid
	}
	return domain.NewDraftContent(title, summary, domainCover, string(raw), tagNames)
}
