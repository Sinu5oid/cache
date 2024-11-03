package cache

import "fmt"

type MissingEntryError struct {
	key string
}

func NewMissingEntryError(key string) MissingEntryError {
	return MissingEntryError{key: key}
}

func (e MissingEntryError) Error() string {
	return fmt.Sprintf("key %s is missing", e.key)
}

type FailedToCastEntryError struct {
	key string
	err error
}

func NewFailedToCastEntryError(key string, err error) FailedToCastEntryError {
	return FailedToCastEntryError{key: key, err: err}
}

func (e FailedToCastEntryError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("could not cast value for key %s: %s", e.key, e.err)
	}

	return fmt.Sprintf("could not cast value for key %s: interface{} could not be casted to output type", e.key)
}
