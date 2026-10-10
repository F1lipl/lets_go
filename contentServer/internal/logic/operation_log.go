package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
)

// logCommittedPostOperation records a business outcome only after its database
// transaction has committed. Request and failure logs are handled at the HTTP
// boundary, so logic does not log the same error a second time.
func logCommittedPostOperation(ctx context.Context, operation, postID, actorID string, fields ...logx.LogField) {
	base := []logx.LogField{
		logx.Field("operation", operation),
		logx.Field("postId", postID),
		logx.Field("actorId", actorID),
	}
	logx.WithContext(ctx).Infow("post operation committed", append(base, fields...)...)
}
