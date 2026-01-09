package domain

import "errors"

// RetriableError defines an interface for errors that can be retried
type RetriableError interface {
	error
	IsRetriable() bool
}

// IsRetriable checks if an error is retriable
func IsRetriable(err error) bool {
	var re RetriableError
	if errors.As(err, &re) {
		return re.IsRetriable()
	}
	return false
}

// NetworkError represents a network-related error that may be retriable
type NetworkError struct {
	Op        string // Operation that failed (e.g., "connect", "read", "write")
	Err       error  // Underlying error
	Retriable bool   // Whether this error is retriable
}

func (e *NetworkError) Error() string {
	return e.Op + ": " + e.Err.Error()
}

func (e *NetworkError) IsRetriable() bool {
	return e.Retriable
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// NewNetworkError creates a new retriable network error
func NewNetworkError(op string, err error) *NetworkError {
	return &NetworkError{Op: op, Err: err, Retriable: true}
}

// NewFatalNetworkError creates a non-retriable network error
func NewFatalNetworkError(op string, err error) *NetworkError {
	return &NetworkError{Op: op, Err: err, Retriable: false}
}

// ConfigError represents a configuration error (never retriable)
type ConfigError struct {
	Field string
	Err   error
}

func (e *ConfigError) Error() string {
	return "config error [" + e.Field + "]: " + e.Err.Error()
}

func (e *ConfigError) IsRetriable() bool {
	return false
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

var (
	// ErrConnectionFailed is returned when websocket connection fails. It's usually retriable.
	ErrConnectionFailed = errors.New("connection failed")

	// ErrInvalidSymbol is returned when a symbol is not supported or malformed. Not retriable.
	ErrInvalidSymbol = errors.New("invalid symbol")

	// ErrUpdateFailed is returned when price update fails
	ErrUpdateFailed = errors.New("update failed")

	// ErrConfigNotFound is returned when configuration file is missing
	ErrConfigNotFound = errors.New("configuration not found")
)
