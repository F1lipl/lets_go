package types

import (
	"errors"
	"strings"
)

const MaxBatchPostCards = 30

var (
	// ErrInvalidBatchPostIDs identifies all invalid batch shapes. The concrete
	// validation reason remains available through errors.Is/errors.Unwrap.
	ErrInvalidBatchPostIDs = errors.New("invalid batch post ids")

	errPostIdsRequired  = errors.New("postIds不能为空")
	errTooManyPostIds   = errors.New("postIds一次最多提交30个")
	errEmptyPostId      = errors.New("postIds不能包含空值")
	errDuplicatePostIds = errors.New("postIds不能包含重复值")
)

// Validate is called by httpx.Parse after decoding the request body.
func (r *BatchGetPostCardsRequest) Validate() error {
	if len(r.PostIds) == 0 {
		return errors.Join(ErrInvalidBatchPostIDs, errPostIdsRequired)
	}
	if len(r.PostIds) > MaxBatchPostCards {
		return errors.Join(ErrInvalidBatchPostIDs, errTooManyPostIds)
	}

	seen := make(map[string]struct{}, len(r.PostIds))
	for _, postId := range r.PostIds {
		postId = strings.TrimSpace(postId)
		if postId == "" {
			return errors.Join(ErrInvalidBatchPostIDs, errEmptyPostId)
		}
		if _, ok := seen[postId]; ok {
			return errors.Join(ErrInvalidBatchPostIDs, errDuplicatePostIds)
		}
		seen[postId] = struct{}{}
	}

	return nil
}
