package example

import (
	"io"
	"time"
)

type Roo interface {
	io.Closer
	V(time.Time)
}
