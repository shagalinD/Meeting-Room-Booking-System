package apperrors

const (
	InternalServerError = iota
	NotFoundError
	UnauthorizedError
	InvalidCredentialsError
	ValidationError
)

type Errors struct {
	Err     error
	Code    int
	Message string
}

func (e *Errors) Error() string {
	return e.Message
}

func (e *Errors) Unwrap() error {
	return e.Err
}