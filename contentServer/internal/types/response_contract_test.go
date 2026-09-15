package types

import (
	"reflect"
	"testing"
)

func TestBusinessResponsesUseEnvelopeAndSeparateData(t *testing.T) {
	tests := []struct {
		response any
		data     any
	}{
		{CreatePostResponse{}, CreatePostData{}},
		{GetPostDraftResponse{}, GetPostDraftData{}},
		{SavePostDraftResponse{}, SavePostDraftData{}},
		{PublishPostResponse{}, PublishPostData{}},
		{ChangePostVisibilityResponse{}, ChangePostVisibilityData{}},
		{DeletePostResponse{}, DeletePostData{}},
		{GetPostResponse{}, GetPostData{}},
		{ListPostsResponse{}, ListPostsData{}},
		{ListMyPostsResponse{}, ListMyPostsData{}},
		{BatchGetPostCardsResponse{}, BatchGetPostCardsData{}},
		{CreateImageUploadResponse{}, CreateImageUploadData{}},
		{CompleteImageUploadResponse{}, CompleteImageUploadData{}},
		{DeleteMediaAssetResponse{}, DeleteMediaAssetData{}},
		{SearchTagsResponse{}, SearchTagsData{}},
	}

	for _, tt := range tests {
		responseType := reflect.TypeOf(tt.response)
		dataType := reflect.TypeOf(tt.data)

		for _, fieldName := range []string{"ErrorCode", "Message", "Data", "RequestId"} {
			if _, ok := responseType.FieldByName(fieldName); !ok {
				t.Errorf("%s is missing %s", responseType.Name(), fieldName)
			}
		}

		dataField, ok := responseType.FieldByName("Data")
		if ok && dataField.Type != dataType {
			t.Errorf("%s.Data type = %s, want %s", responseType.Name(), dataField.Type, dataType)
		}

		for _, fieldName := range []string{"ErrorCode", "Message", "RequestId"} {
			if _, ok := dataType.FieldByName(fieldName); ok {
				t.Errorf("%s must not contain transport field %s", dataType.Name(), fieldName)
			}
		}
	}
}
