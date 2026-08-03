package handler

type errorResponse struct {
	Error string `json:"error" example:"error message"`
}

type messageResponse struct {
	Message string `json:"message" example:"ok"`
}
