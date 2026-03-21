package dto

type TagResponse struct {
	Name      string `json:"name"`
	PostCount int64  `json:"post_count"`
}

type TagListResponse struct {
	Tags  []TagResponse `json:"tags"`
	Total int64         `json:"total"`
}
