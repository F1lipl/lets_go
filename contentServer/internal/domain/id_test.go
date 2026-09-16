package domain

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestParsePostID(t *testing.T) {
	want := uuid.New()
	id, err := ParsePostID(want.String())
	if err != nil {
		t.Fatalf("ParsePostID() error = %v", err)
	}
	if got := id.String(); got != want.String() {
		t.Fatalf("String() = %q, want %q", got, want.String())
	}
}

func TestParsePostIDRejectsInvalidAndNilUUID(t *testing.T) {
	for _, value := range []string{"not-a-uuid", uuid.Nil.String()} {
		_, err := ParsePostID(value)
		if !errors.Is(err, ErrInvalidPostID) {
			t.Fatalf("ParsePostID(%q) error = %v, want ErrInvalidPostID", value, err)
		}
	}
}

func TestPostIDBytesReturnsCopy(t *testing.T) {
	id, err := ParsePostID(uuid.New().String())
	if err != nil {
		t.Fatalf("ParsePostID() error = %v", err)
	}

	first := id.Bytes()
	first[0] ^= 0xff
	second := id.Bytes()
	if first[0] == second[0] {
		t.Fatal("Bytes() exposed mutable identifier storage")
	}
}

func TestIDClonePreservesValue(t *testing.T) {
	postID, err := NewPostID()
	if err != nil {
		t.Fatal(err)
	}
	userID, err := ParseUserID(uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	revisionID, err := NewRevisionID()
	if err != nil {
		t.Fatal(err)
	}
	assetID, err := NewAssetID()
	if err != nil {
		t.Fatal(err)
	}

	if cloned := postID.Clone(); cloned != postID {
		t.Fatalf("PostID.Clone() = %v, want %v", cloned, postID)
	}
	if cloned := userID.Clone(); cloned != userID {
		t.Fatalf("UserID.Clone() = %v, want %v", cloned, userID)
	}
	if cloned := revisionID.Clone(); cloned != revisionID {
		t.Fatalf("RevisionID.Clone() = %v, want %v", cloned, revisionID)
	}
	if cloned := assetID.Clone(); cloned != assetID {
		t.Fatalf("AssetID.Clone() = %v, want %v", cloned, assetID)
	}
}

func TestAssetIDJSONRoundTrip(t *testing.T) {
	want, err := NewAssetID()
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}

	var got AssetID
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("AssetID JSON round trip = %v, want %v", got, want)
	}
}
