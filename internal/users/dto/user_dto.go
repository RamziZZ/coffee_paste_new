package dto

type UpdateMyProfileRequest struct {
	Name     string  `json:"name" binding:"required"`
	Phone    *string `json:"phone"`
	PhotoURL *string `json:"photo_url"`
}

type UpdateUserStatusRequest struct {
	IsActive bool `json:"is_active"`
}
