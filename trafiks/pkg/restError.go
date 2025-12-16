package pkg

import "net/http"

type RestErrClient interface {
	BadRequest(message string) *RestErr
	Unauthorized(message string) *RestErr
	NotFound(message string) *RestErr
	ServerError(message string) *RestErr
	RequestNotAllowed(message string) *RestErr
	StatusConflict(message string) *RestErr
	BadRequestWithAction(message, action string) *RestErr
}

type RestErr struct {
	Message    string `json:"message"`
	Success    bool   `json:"success"`
	StatusCode int    `json:"code"`
	Action     string `json:"action,omitempty"`
}

func (r *RestErr) BadRequest(message string) *RestErr {
	return &RestErr{
		Message:    message,
		Success:    false,
		StatusCode: http.StatusBadRequest,
	}
}

func (r *RestErr) BadRequestWithAction(message, action string) *RestErr {
	return &RestErr{
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Action:     action,
	}
}

func (r *RestErr) Unauthorized(message string) *RestErr {
	return &RestErr{
		Message:    message,
		Success:    false,
		StatusCode: http.StatusUnauthorized,
	}
}

func (r *RestErr) NotFound(message string) *RestErr {
	return &RestErr{
		Message:    message,
		Success:    false,
		StatusCode: http.StatusNotFound,
	}
}

func (r *RestErr) ServerError(message string) *RestErr {
	return &RestErr{
		Message:    message,
		Success:    false,
		StatusCode: http.StatusInternalServerError,
	}
}

func (r *RestErr) RequestNotAllowed(message string) *RestErr {
	return &RestErr{
		Message:    message,
		Success:    false,
		StatusCode: http.StatusForbidden,
	}
}

func (r *RestErr) StatusConflict(message string) *RestErr {
	return &RestErr{
		Message:    message,
		Success:    true,
		StatusCode: http.StatusConflict,
	}
}

func NewRestErr() RestErrClient {
	return &RestErr{}
}
