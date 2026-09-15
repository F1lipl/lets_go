package domain

import (
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
