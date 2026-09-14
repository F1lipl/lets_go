package types

import (
	"errors"
	"strings"
)

const MaxBatchPostCards = 30

var (
	errPostIdsRequired  = errors.New("postIds不能为空")
	errTooManyPostIds   = errors.New("postIds一次最多提交30个")
	errEmptyPostId      = errors.New("postIds不能包含空值")
	errDuplicatePostIds = errors.New("postIds不能包含重复值")
)

// Validate is called by httpx.Parse after decoding the request body.
func (r *BatchGetPostCardsRequest) Validate() error {
	if len(r.PostIds) == 0 {
		return errPostIdsRequired
	}
	if len(r.PostIds) > MaxBatchPostCards {
		return errTooManyPostIds
	}

	seen := make(map[string]struct{}, len(r.PostIds))
	for _, postId := range r.PostIds {
		postId = strings.TrimSpace(postId)
		if postId == "" {
			return errEmptyPostId
		}
		if _, ok := seen[postId]; ok {
			return errDuplicatePostIds
		}
		seen[postId] = struct{}{}
	}

	return nil
}
