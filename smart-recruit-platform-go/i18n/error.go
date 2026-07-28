package i18n

import "errors"

type KeyError struct {
	Key   string
	Args  Args
	Cause error
}

func NewError(key string, args ...Args) *KeyError {
	return &KeyError{Key: key, Args: mergeArgs(args)}
}

func Wrap(key string, cause error, args ...Args) *KeyError {
	return &KeyError{Key: key, Args: mergeArgs(args), Cause: cause}
}

func (err *KeyError) Error() string {
	if err == nil {
		return T("common.unknown_error")
	}
	return T(err.Key, err.Args)
}

func (err *KeyError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

func ErrorKey(err error) (string, Args, bool) {
	var keyed *KeyError
	if errors.As(err, &keyed) {
		return keyed.Key, keyed.Args, true
	}
	return "", nil, false
}
