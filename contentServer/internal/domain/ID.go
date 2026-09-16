package domain

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type PostID struct {
	value uuid.UUID
}

func NewPostID() (PostID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return PostID{}, err
	}

	return PostID{value: id}, nil
}

func ParsePostID(value string) (PostID, error) {
	id, err := parseUUID(value, ErrInvalidPostID)
	if err != nil {
		return PostID{}, err
	}

	return PostID{value: id}, nil
}

func (id PostID) String() string {
	return id.value.String()
}

// Clone returns an independent value copy of id.
func (id PostID) Clone() PostID {
	return PostID{value: id.value}
}

func (id PostID) IsZero() bool {
	return id.value == uuid.Nil
}

func (id PostID) Bytes() []byte {
	return uuidBytes(id.value)
}

// UserID is issued by userServer. contentServer only parses and stores it; it
// must not generate a replacement identity for a caller.
type UserID struct {
	value uuid.UUID
}

func ParseUserID(value string) (UserID, error) {
	id, err := parseUUID(value, ErrInvalidUserID)
	if err != nil {
		return UserID{}, err
	}

	return UserID{value: id}, nil
}

func (id UserID) String() string {
	return id.value.String()
}

// Clone returns an independent value copy of id.
func (id UserID) Clone() UserID {
	return UserID{value: id.value}
}

func (id UserID) IsZero() bool {
	return id.value == uuid.Nil
}

func (id UserID) Bytes() []byte {
	return uuidBytes(id.value)
}

type RevisionID struct {
	value uuid.UUID
}

func NewRevisionID() (RevisionID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return RevisionID{}, err
	}

	return RevisionID{value: id}, nil
}

func ParseRevisionID(value string) (RevisionID, error) {
	id, err := parseUUID(value, ErrInvalidRevisionID)
	if err != nil {
		return RevisionID{}, err
	}

	return RevisionID{value: id}, nil
}

func (id RevisionID) String() string {
	return id.value.String()
}

// Clone returns an independent value copy of id.
func (id RevisionID) Clone() RevisionID {
	return RevisionID{value: id.value}
}

func (id RevisionID) IsZero() bool {
	return id.value == uuid.Nil
}

func (id RevisionID) Bytes() []byte {
	return uuidBytes(id.value)
}

func uuidBytes(id uuid.UUID) []byte {
	value := make([]byte, len(id))
	copy(value, id[:])
	return value
}

type AssetID struct {
	value uuid.UUID
}

func NewAssetID() (AssetID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return AssetID{}, err
	}

	return AssetID{value: id}, nil
}

func ParseAssetID(value string) (AssetID, error) {
	id, err := parseUUID(value, ErrInvalidAssetID)
	if err != nil {
		return AssetID{}, err
	}

	return AssetID{value: id}, nil
}

func (id AssetID) String() string {
	return id.value.String()
}

// Clone returns an independent value copy of id.
func (id AssetID) Clone() AssetID {
	return AssetID{value: id.value}
}

func (id AssetID) IsZero() bool {
	return id.value == uuid.Nil
}

func (id AssetID) Bytes() []byte {
	return uuidBytes(id.value)
}

func (id AssetID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.value.String())
}

func (id *AssetID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidAssetID, err)
	}

	parsed, err := ParseAssetID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

func parseUUID(value string, invalidError error) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %v", invalidError, err)
	}
	if id == uuid.Nil {
		return uuid.Nil, invalidError
	}

	return id, nil
}
