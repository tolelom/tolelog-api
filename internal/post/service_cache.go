package post

import (
	"fmt"
	"log/slog"
	"time"
	"tolelom_api/internal/model"
)

// Cache key patterns and TTLs
const (
	cachePublicPosts = "posts:public:%d:%d:%s" // page, pageSize, tag
	cachePost        = "posts:%d"              // postID
	cacheTTLList     = 2 * time.Minute
	cacheTTLPost     = 5 * time.Minute
)

// cachedPublicPostList is the cached structure for public post list responses.
type cachedPublicPostList struct {
	Posts []model.Post `json:"posts"`
	Total int64        `json:"total"`
}

// invalidatePostCaches invalidates list caches and optionally a specific post cache.
func (s *service) invalidatePostCaches(postID uint) {
	if s.cache == nil {
		return
	}
	if err := s.cache.DeleteByPattern("posts:public:*"); err != nil {
		slog.Warn("캐시 삭제 실패 (목록)", "error", err)
	}
	if postID > 0 {
		if err := s.cache.Delete(fmt.Sprintf(cachePost, postID)); err != nil {
			slog.Warn("캐시 삭제 실패 (개별 글)", "postID", postID, "error", err)
		}
	}
}
