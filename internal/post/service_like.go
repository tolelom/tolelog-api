package post

import (
	"errors"
	"tolelom_api/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *service) ToggleLike(postID uint, userID uint) (liked bool, likeCount uint, err error) {
	target, err := s.findPost(s.db.Select("id, user_id, is_public, status"), postID)
	if err != nil {
		return false, 0, err
	}
	// 볼 수 없는 글은 없는 글과 같게 다뤄 존재를 드러내지 않는다.
	if !target.VisibleTo(&userID) {
		return false, 0, ErrPostNotFound
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		var existing model.PostLike
		findErr := tx.Where("post_id = ? AND user_id = ?", postID, userID).First(&existing).Error
		switch {
		case findErr == nil:
			liked = false
			if err := removeLike(tx, &existing); err != nil {
				return err
			}
		case errors.Is(findErr, gorm.ErrRecordNotFound):
			liked = true
			if err := addLike(tx, postID, userID); err != nil {
				return err
			}
		default:
			return findErr
		}

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

// removeLike·addLike 는 같은 사용자의 겹친 요청으로 읽은 상태가 쓰기 전에 바뀔 수 있어,
// 행이 실제로 바뀐 경우에만 like_count 를 움직인다.
func removeLike(tx *gorm.DB, like *model.PostLike) error {
	res := tx.Delete(like)
	if res.Error != nil || res.RowsAffected == 0 {
		return res.Error
	}
	return tx.Model(&model.Post{}).Where("id = ?", like.PostID).
		UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")).Error
}

func addLike(tx *gorm.DB, postID, userID uint) error {
	res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.PostLike{PostID: postID, UserID: userID})
	if res.Error != nil || res.RowsAffected == 0 {
		return res.Error
	}
	return tx.Model(&model.Post{}).Where("id = ?", postID).
		UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
}

func (s *service) IsLiked(postID uint, userID uint) (bool, error) {
	var count int64
	if err := s.db.Model(&model.PostLike{}).Where("post_id = ? AND user_id = ?", postID, userID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
