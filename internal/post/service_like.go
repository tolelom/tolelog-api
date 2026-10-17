package post

import (
	"errors"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ToggleLike toggles a like for a post. Returns the new liked state and total like count.
func (s *service) ToggleLike(postID uint, userID uint) (bool, uint, error) {
	var target model.Post
	if err := s.db.Select("id, user_id, is_public, status").First(&target, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, 0, ErrPostNotFound
		}
		return false, 0, err
	}
	// 볼 수 없는 글은 없는 글과 같게 다뤄 존재를 드러내지 않는다.
	if !target.VisibleTo(&userID) {
		return false, 0, ErrPostNotFound
	}

	var liked bool
	var likeCount uint

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var existing model.PostLike
		findErr := tx.Where("post_id = ? AND user_id = ?", postID, userID).First(&existing).Error

		// 같은 사용자의 요청이 겹치면 읽은 상태가 쓰기 전에 바뀔 수 있다. 행이 실제로
		// 바뀐 경우에만 like_count 를 움직인다.
		if findErr == nil {
			// Already liked → unlike
			res := tx.Delete(&existing)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 1 {
				if err := tx.Model(&model.Post{}).Where("id = ?", postID).
					UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")).Error; err != nil {
					return err
				}
			}
			liked = false
		} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
			// Not liked → like. 이미 다른 요청이 넣었으면 좋아요 상태로 본다.
			like := model.PostLike{PostID: postID, UserID: userID}
			res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&like)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 1 {
				if err := tx.Model(&model.Post{}).Where("id = ?", postID).
					UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
					return err
				}
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

	return liked, likeCount, nil
}

// IsLiked checks if a user has liked a post.
func (s *service) IsLiked(postID uint, userID uint) (bool, error) {
	var count int64
	if err := s.db.Model(&model.PostLike{}).Where("post_id = ? AND user_id = ?", postID, userID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
