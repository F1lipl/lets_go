package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestPrepareTagNamesNormalizesAndDeduplicates(t *testing.T) {
	names, err := PrepareTagNames([]string{" #Travel ", "travel", "＃城市", "#城市"})
	if err == nil {
		t.Fatal("unsupported embedded fullwidth marker accepted")
	}
	names, err = PrepareTagNames([]string{" #Travel ", "travel", "#城市"})
	if err != nil || len(names) != 2 || names[0].Normalized != "travel" || names[0].Display != "Travel" || names[1].Display != "城市" {
		t.Fatalf("prepared names = %+v, err = %v", names, err)
	}
}

func TestPrepareTagNamesRejectsInvalidInput(t *testing.T) {
	if _, err := PrepareTagNames([]string{"#"}); !errors.Is(err, ErrTagNameInvalid) {
		t.Fatalf("empty name: %v", err)
	}
	if _, err := PrepareTagNames([]string{strings.Repeat("a", 65)}); !errors.Is(err, ErrTagNameInvalid) {
		t.Fatalf("long name: %v", err)
	}
	if _, err := PrepareTagNames(make([]string, MaxPostTags+1)); !errors.Is(err, ErrTooManyTags) {
		t.Fatalf("too many names: %v", err)
	}
}
