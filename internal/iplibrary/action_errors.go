package iplibrary

// FataError 是否是致命错误
type FataError struct {
	err string
}

func (this *FataError) Error() string {
	return this.err
}

func NewFataError(err string) error {
	return &FataError{err: err}
}

func IsFatalError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*FataError)
	return ok
}
