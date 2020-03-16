package errors

type CustomLogoError struct {
	message string
}

// implement the error interface
func (e *CustomLogoError) Error() string {
	return e.message
}

func NewCustomLogoError(msg string) *CustomLogoError {
	err := &CustomLogoError{
		message: msg,
	}
	return err
}

func IsCustomLogoError(err error) bool {
	_, ok := err.(*CustomLogoError)
	return ok
}
