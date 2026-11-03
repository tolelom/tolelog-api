package user

import (
	"errors"
	"strings"
	"time"
	"tolelom_api/internal/auth"
	"tolelom_api/internal/dto"
	"tolelom_api/internal/model"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidPassword     = errors.New("invalid current password")
	ErrSamePassword        = errors.New("new password is the same as current")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

// 앞선 중복 확인과 INSERT 사이에 같은 이름으로 가입한 요청은 유니크 인덱스에 걸린다.
const mysqlErrDuplicateEntry = 1062

type AuthService interface {
	RegisterUser(req *dto.RegisterRequest) (*dto.AuthResponse, error)
	AuthenticateUser(req *dto.LoginRequest) (*dto.AuthResponse, error)
	GetUserByID(userID uint) (*model.User, error)
	UpdateAvatar(userID uint, avatarURL string) error
	RefreshTokens(refreshToken string) (*dto.AuthResponse, error)
	ChangePassword(userID uint, req *dto.ChangePasswordRequest) (*dto.AuthResponse, error)
	Logout(userID uint) error
	DeleteUser(userID uint) error
}

type authService struct {
	db        *gorm.DB
	jwtSecret string
}

func NewAuthService(db *gorm.DB, jwtSecret string) AuthService {
	return &authService{db: db, jwtSecret: jwtSecret}
}

func (s *authService) RegisterUser(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	req.Username = normalizeUsername(req.Username)

	var existing model.User
	if err := s.db.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return nil, ErrUserAlreadyExists
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	u := &model.User{
		Username:     req.Username,
		PasswordHash: hash,
		LastLogin:    time.Now(),
	}

	if err := s.db.Create(u).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrDuplicateEntry {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	return s.issueTokens(u)
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func (s *authService) issueTokens(u *model.User) (*dto.AuthResponse, error) {
	accessToken, refreshToken, err := auth.GenerateTokenPair(u, s.jwtSecret)
	if err != nil {
		return nil, err
	}
	return &dto.AuthResponse{
		User:         dto.UserToResponse(u),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *authService) GetUserByID(userID uint) (*model.User, error) {
	var u model.User
	if err := s.db.First(&u, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (s *authService) AuthenticateUser(req *dto.LoginRequest) (*dto.AuthResponse, error) {
	req.Username = normalizeUsername(req.Username)

	var u model.User
	if err := s.db.Where("username = ?", req.Username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if !auth.CheckPasswordHash(req.Password, u.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// last_login 칸만 쓴다. Save 는 읽어 둔 행 전체를 다시 써서, 그 사이 커밋된 로그아웃·비밀번호 변경의
	// token_version·password 를 되돌린다.
	u.LastLogin = time.Now()
	if err := s.db.Model(&u).UpdateColumn("last_login", u.LastLogin).Error; err != nil {
		return nil, err
	}

	return s.issueTokens(&u)
}

func (s *authService) UpdateAvatar(userID uint, avatarURL string) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).Update("avatar_url", avatarURL).Error
}

func (s *authService) RefreshTokens(refreshToken string) (*dto.AuthResponse, error) {
	claims, err := auth.ValidateRefreshToken(refreshToken, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	u, err := s.GetUserByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	if claims.TokenVersion != u.TokenVersion {
		return nil, ErrInvalidRefreshToken
	}

	return s.issueTokens(u)
}

func (s *authService) Logout(userID uint) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).UpdateColumn("token_version", gorm.Expr("token_version + 1")).Error
}

// DeleteUser 는 사용자를 지운다. 글·시리즈·댓글은 외래 키 CASCADE 로 함께 지워지지만, post_tags 는 글을 막고
// post_likes 에는 외래 키가 없어 먼저 정리한다.
func (s *authService) DeleteUser(userID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		ownPostIDs := tx.Unscoped().Model(&model.Post{}).Select("id").Where("user_id = ?", userID)

		if err := tx.Exec("UPDATE posts SET like_count = like_count - 1 WHERE like_count > 0 AND id IN (SELECT post_id FROM post_likes WHERE user_id = ?)", userID).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? OR post_id IN (?)", userID, ownPostIDs).Delete(&model.PostLike{}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM post_tags WHERE post_id IN (?)", ownPostIDs).Error; err != nil {
			return err
		}
		return tx.Delete(&model.User{}, userID).Error
	})
}

// ChangePassword 는 token_version 을 올려 다른 기기의 리프레시 토큰을 끊고, 바꾼 기기에는 새 토큰 쌍을 준다.
func (s *authService) ChangePassword(userID uint, req *dto.ChangePasswordRequest) (*dto.AuthResponse, error) {
	u, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	if !auth.CheckPasswordHash(req.CurrentPassword, u.PasswordHash) {
		return nil, ErrInvalidPassword
	}

	if auth.CheckPasswordHash(req.NewPassword, u.PasswordHash) {
		return nil, ErrSamePassword
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return nil, err
	}

	if err := s.db.Model(u).Updates(map[string]any{
		"password":      hash,
		"token_version": gorm.Expr("token_version + 1"),
	}).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(u, userID).Error; err != nil {
		return nil, err
	}

	return s.issueTokens(u)
}
