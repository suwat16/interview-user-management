package handler

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}
