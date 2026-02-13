package dto

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

type AuthDataResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	UserID   uint   `json:"user_id"`
}
