package types

import (
	"reflect"
	"testing"
)

func TestBusinessResponsesExposeCommonResultFields(t *testing.T) {
	responses := []any{
		&CreatePostResponse{},
		&GetPostDraftResponse{},
		&SavePostDraftResponse{},
		&CreatePostRouteDraftResponse{},
		&DetachPostRouteDraftResponse{},
		&PublishPostResponse{},
		&ChangePostVisibilityResponse{},
		&DeletePostResponse{},
		&GetPostResponse{},
		&ListPostsResponse{},
		&ListMyPostsResponse{},
		&BatchGetPostCardsResponse{},
		&CreateImageUploadResponse{},
		&CompleteImageUploadResponse{},
		&DeleteMediaAssetResponse{},
		&SearchTagsResponse{},
	}

	for _, response := range responses {
		typ := reflect.TypeOf(response).Elem()
		for _, fieldName := range []string{"ErrorCode", "Message"} {
			if _, ok := typ.FieldByName(fieldName); !ok {
				t.Errorf("%s is missing %s", typ.Name(), fieldName)
			}
		}

		if _, ok := response.(interface {
			SetResult(errorCode int, message string)
		}); !ok {
			t.Errorf("%s cannot receive common result fields", typ.Name())
		}
	}
}
