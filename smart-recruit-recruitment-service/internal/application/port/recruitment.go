package port

import "time"

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

type UploadIDGenerator interface {
	NewUploadID() (string, error)
}

type UploadIDFunc func() (string, error)

func (f UploadIDFunc) NewUploadID() (string, error) { return f() }
