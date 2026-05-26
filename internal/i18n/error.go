package i18n

type LocalizedError struct {
	Key   string
	Args  map[string]any
	Cause error
}

func NewError(key string, args map[string]any) *LocalizedError {
	return &LocalizedError{Key: key, Args: args}
}

func WrapError(key string, args map[string]any, cause error) *LocalizedError {
	return &LocalizedError{Key: key, Args: args, Cause: cause}
}

func (e *LocalizedError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Key
}

func (e *LocalizedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}
