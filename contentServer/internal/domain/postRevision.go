package domain

import "time"

type PostRevision struct {
	RevisionId     RevisionID
	PostId         PostID
	RevisionNumber uint64
	Title          string
	Summary        string
	Cover          *Cover
	Document       PostDocument
	CreatedAt      time.Time
	PublishedAt    time.Time
}

func CreateNewPostRevision(id RevisionID, postID PostID, revisionNumber uint64, title string, summary string, cover *Cover, document PostDocument, now time.Time) *PostRevision {
	return &PostRevision{
		RevisionId:     id,
		PostId:         postID,
		RevisionNumber: revisionNumber,
		Title:          title,
		Summary:        summary,
		Cover:          cover,
		Document:       document,
		CreatedAt:      now,
		PublishedAt:    now,
	}
}
