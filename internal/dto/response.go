package dto

type ErrorResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

func NewErrorResponse(errCode, message string) ErrorResponse {
	return ErrorResponse{
		Status:  "error",
		Error:   errCode,
		Message: message,
	}
}

type AuthDataResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Username     string `json:"username"`
	UserID       uint   `json:"user_id"`
	AvatarURL    string `json:"avatar_url"`
}

// AuthDataFrom flattens an AuthResponse into the shape auth endpoints return.
func AuthDataFrom(r *AuthResponse) AuthDataResponse {
	return AuthDataResponse{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		Username:     r.User.Username,
		UserID:       r.User.ID,
		AvatarURL:    r.User.AvatarURL,
	}
}
