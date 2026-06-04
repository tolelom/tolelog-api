package post

import (
	"errors"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
)

// ToggleLike toggles a like for a post. Returns the new liked state and total like count.
func (s *service) ToggleLike(postID uint, userID uint) (bool, uint, error) {
	var liked bool
	var likeCount uint

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var existing model.PostLike
		findErr := tx.Where("post_id = ? AND user_id = ?", postID, userID).First(&existing).Error

		if findErr == nil {
			// Already liked → unlike
			if err := tx.Delete(&existing).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Post{}).Where("id = ? AND like_count > 0", postID).
				UpdateColumn("like_count", gorm.Expr("like_count - 1")).Error; err != nil {
				return err
			}
			liked = false
		} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
			// Not liked → like
			like := model.PostLike{PostID: postID, UserID: userID}
			if err := tx.Create(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Post{}).Where("id = ?", postID).
				UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
				return err
			}
			liked = true
		} else {
			return findErr
		}

		// Get updated count within transaction
		var post model.Post
		if err := tx.Select("like_count").First(&post, postID).Error; err != nil {
			return err
		}
		likeCount = post.LikeCount
		return nil
	})
	if err != nil {
		return false, 0, err
	}

	s.invalidatePostCaches(postID)

	return liked, likeCount, nil
}

// IsLiked checks if a user has liked a post.
func (s *service) IsLiked(postID uint, userID uint) bool {
	var count int64
	s.db.Model(&model.PostLike{}).Where("post_id = ? AND user_id = ?", postID, userID).Count(&count)
	return count > 0
}
