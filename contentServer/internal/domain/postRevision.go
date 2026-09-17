package domain

import "time"

type PostRevision struct {
	RevisionId            RevisionID
	PostId                PostID
	RevisionNumber        uint64
	SourceDraftVersion    uint64
	DocumentSchemaVersion uint64
	PlainText             string
	BlockCount            uint64
	ImageCount            uint64
	TagNames              []string
	Title                 string
	Summary               string
	Cover                 *Cover
	Document              string
	PublishedAt           time.Time
}

func CreateNewPostRevision(id RevisionID, postID PostID, revisionNumber uint64, title string, summary string, cover *Cover, document string, now time.Time) *PostRevision {
	if cover != nil {
		cover = new(*cover)
	}
	return &PostRevision{
		RevisionId:     id,
		PostId:         postID,
		RevisionNumber: revisionNumber,
		Title:          title,
		Summary:        summary,
		Cover:          cover,
		Document:       document,
		PublishedAt:    now,
	}
}
