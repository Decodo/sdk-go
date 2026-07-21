package decodo

// DecodoError is the base error type for all Decodo API errors.
type DecodoError struct {
	StatusCode int
	APIStatus  string
	Message    string
}

func (e *DecodoError) Error() string {
	return e.Message
}

// AuthenticationError is returned on 401/403 responses.
type AuthenticationError struct {
	DecodoError
}

// RateLimitError is returned on 429 responses.
type RateLimitError struct {
	DecodoError
}

// ValidationError is returned on 422/400 responses with validation errors.
type ValidationError struct {
	DecodoError
	Errors []interface{}
}

// TimeoutError is returned when a request times out.
type TimeoutError struct {
	Msg string
}

func (e *TimeoutError) Error() string {
	return e.Msg
}

func mapHTTPError(statusCode int, message, apiStatus string, errors []interface{}) error {
	base := DecodoError{StatusCode: statusCode, APIStatus: apiStatus, Message: message}
	switch {
	case statusCode == 401 || statusCode == 403:
		return &AuthenticationError{base}
	case statusCode == 429:
		return &RateLimitError{base}
	case statusCode == 422 || (statusCode == 400 && len(errors) > 0):
		return &ValidationError{DecodoError: base, Errors: errors}
	default:
		return &base
	}
}
