package types

import (
	"fmt"
	"testing"
)

func TestBatchGetPostCardsRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		postIds []string
		wantErr bool
	}{
		{name: "one id", postIds: []string{"post-1"}},
		{name: "thirty ids", postIds: makePostIds(MaxBatchPostCards)},
		{name: "missing ids", wantErr: true},
		{name: "too many ids", postIds: makePostIds(MaxBatchPostCards + 1), wantErr: true},
		{name: "empty id", postIds: []string{"post-1", " "}, wantErr: true},
		{name: "duplicate id", postIds: []string{"post-1", "post-1"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := BatchGetPostCardsRequest{PostIds: tt.postIds}
			if err := req.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func makePostIds(count int) []string {
	ids := make([]string, count)
	for i := range ids {
		ids[i] = fmt.Sprintf("post-%d", i)
	}
	return ids
}
