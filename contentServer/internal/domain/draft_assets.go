package domain

type DraftMediaAssetState struct {
	ID        AssetID
	OwnerID   UserID
	Status    uint64
	IsDeleted bool
}

func (draft *PostDraft) ReferencedAssetIDs() []AssetID {
	ids := make([]AssetID, 0, len(draft.AssetRefs)+1)
	seen := make(map[AssetID]bool, cap(ids))
	add := func(id AssetID) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if draft.Cover != nil {
		add(draft.Cover.AssetID)
	}
	for _, ref := range draft.AssetRefs {
		add(ref.AssetID)
	}
	return ids
}

// Drafts may refer to an image while it is uploading or processing. Publishing
// still requires the ready state, checked separately in the publish workflow.
func (draft *PostDraft) ValidateAssetStates(authorID UserID, states []DraftMediaAssetState) error {
	expected := make(map[AssetID]bool)
	for _, id := range draft.ReferencedAssetIDs() {
		expected[id] = true
	}
	if len(states) != len(expected) {
		return ErrMediaAssetNotFound
	}
	for _, state := range states {
		if !expected[state.ID] || state.OwnerID != authorID {
			return ErrMediaAssetNotFound
		}
		if state.IsDeleted || state.Status < 1 || state.Status > 3 {
			return ErrMediaAssetNotReady
		}
		delete(expected, state.ID)
	}
	if len(expected) != 0 {
		return ErrMediaAssetNotFound
	}
	return nil
}
