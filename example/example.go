package example

import (
	"github.com/0LuigiCode0/tshort/example/test1"
)

// go:generate mockery --name Doo --outpkg=mockmain
//
//go:generate tshort --name Doo
type (
	Doo interface {
		A(*int, int, []byte) (test1.INT, error)
		B()
	}
)

// тестируемая функция
func Foo(a *int, boo Doo) (b test1.INT, err error) {
	if *a%2 == 0 {
		if b, err = boo.A(a, *a, []byte{}); err != nil {
			return
		}
		*a *= 5
	} else {
		*a--
		if b, err = boo.A(a, *a, []byte{}); err != nil {
			return
		}
	}
	boo.B()
	return
}
