package domain

import "time"

type PostRevision struct {
	RevisionId     RevisionID
	PostId         PostID
	RevisionNumber uint64
	Title          string
	Summary        string
	Cover          *Cover
	Document       string
	PublishedAt    time.Time
}

func CreateNewPostRevision(id RevisionID, postID PostID, revisionNumber uint64, title string, summary string, cover *Cover, document string, now time.Time) *PostRevision {
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
