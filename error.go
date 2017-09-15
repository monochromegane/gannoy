package gannoy

type NGTSearchError struct {
	message string
}

func (err NGTSearchError) Error() string {
	return err.message
}

func newNGTSearchErrorFrom(err error) error {
	if err == nil {
		return nil
	} else {
		return NGTSearchError{message: err.Error()}
	}
}

type TimeoutError struct {
	message string
}

func (err TimeoutError) Error() string {
	return err.message
}

func newTimeoutErrorFrom(err error) error {
	if err == nil {
		return nil
	} else {
		return TimeoutError{message: err.Error()}
	}
}
