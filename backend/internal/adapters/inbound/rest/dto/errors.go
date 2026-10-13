package dto

// ErrorResponse es el formato único de error de la API.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody describe el error con un código estable para los clientes.
type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// NewError construye una respuesta de error.
func NewError(code, message string, details map[string]string) ErrorResponse {
	return ErrorResponse{Error: ErrorBody{Code: code, Message: message, Details: details}}
}
