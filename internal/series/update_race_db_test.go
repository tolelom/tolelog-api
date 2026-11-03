package series

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"
	"tolelom_api/internal/testutil"
)

// 수정 요청이 시리즈를 읽은 뒤 쓰기 직전에, 같은 작성자의 삭제 요청이 커밋된 상황을 만든다.
func TestUpdateSeries_DoesNotResurrectConcurrentlyDeletedSeries(t *testing.T) {
	db := testutil.SetupDB(t)
	svc := NewService(db)
	u := testutil.MakeUser(t, db, "alice")
	s := &model.Series{Title: "원래 제목", UserID: u.ID}
	if err := db.Create(s).Error; err != nil {
		t.Fatal(err)
	}

	fired := false
	err := db.Callback().Update().Before("gorm:update").Register("test:concurrent_delete", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != "series" {
			return
		}
		fired = true
		tx.Session(&gorm.Session{NewDB: true}).
			Exec("UPDATE series SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", s.ID)
	})
	if err != nil {
		t.Fatal(err)
	}

	title := "고친 제목"
	_, err = svc.UpdateSeries(s.ID, u.ID, &dto.UpdateSeriesRequest{Title: &title})
	if !fired {
		t.Fatal("겹친 삭제가 주입되지 않았다")
	}
	if !errors.Is(err, ErrSeriesNotFound) {
		t.Errorf("err = %v, want ErrSeriesNotFound", err)
	}
	var visible int64
	db.Model(&model.Series{}).Where("id = ?", s.ID).Count(&visible)
	if visible != 0 {
		t.Fatal("삭제된 시리즈가 수정 요청으로 되살아났다")
	}
}
