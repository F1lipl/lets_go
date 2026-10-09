package domain

type PublishedAsset struct {
	ID       AssetID
	MimeType string
	Width    int64
	Height   int64
}

type PublishedTag struct {
	ID   string
	Name string
}

type PostCounts struct {
	Likes     uint64
	Favorites uint64
	Comments  uint64
}

// CurrentPublicRevision selects the only version that a public read may display.
// The Post is authoritative; an asynchronously maintained card is not consulted.
func (post *Post) CurrentPublicRevision() (RevisionID, error) {
	if post == nil || post.LifecycleStatus == LifecycleDeleted || post.DeletedAt != nil {
		return RevisionID{}, ErrPostNotFound
	}
	if post.LifecycleStatus != LifecyclePublished || post.PublishedRevisionID == nil {
		return RevisionID{}, ErrPostNotPublished
	}
	if post.Visibility != VisibilityPublic || post.AvailabilityStatus != AvailabilityNormal {
		return RevisionID{}, ErrPostNotVisible
	}
	return *post.PublishedRevisionID, nil
}

func (revision *PostRevision) BelongsTo(postID PostID) bool {
	return revision != nil && revision.PostId == postID
}
