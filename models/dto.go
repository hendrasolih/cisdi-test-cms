package models

type RegisterRequest struct {
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=6"`
	Role     UserRole `json:"role,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreateArticleRequest struct {
	Title   string   `json:"title" binding:"required,min=1,max=255"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

type CreateArticleVersionRequest struct {
	Title   string   `json:"title" binding:"required,min=1,max=255"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

type UpdateVersionStatusRequest struct {
	Status VersionStatus `json:"status" binding:"required"`
}

type CreateTagRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

type ArticleListParams struct {
	Status    string `form:"status"`
	AuthorID  uint   `form:"author_id"`
	TagID     uint   `form:"tag_id"`
	Page      int    `form:"page,default=1"`
	Limit     int    `form:"limit,default=10"`
	SortBy    string `form:"sort_by,default=created_at"`
	SortOrder string `form:"sort_order,default=desc"`
}
