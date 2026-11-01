package example

import (
	"errors"
	"testing"

	"github.com/0LuigiCode0/tshort"
	"github.com/0LuigiCode0/tshort/example/mocks"
	"github.com/0LuigiCode0/tshort/example/test1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var deferr = errors.New("hello")

func TestFoo(t *testing.T) {
	// объявление моковых классов и входящих данных: переменные в скоупе
	// теста, захвачены замыканиями CASE/BREAK
	var m *mocks.MockDoo
	var a *int
	var a2 *int

	tshort.Run(t, func(c *tshort.Collector[test1.INT]) {
		ctrl := gomock.NewController(c.T)
		m = mocks.NewMockDoo(ctrl)
		a = new(int)
		a2 = new(int)

		c.CASE(func(c *tshort.Collector[test1.INT]) {
			*a = 4

			c.BREAK("Test 1: четное error", func(c *tshort.Case[test1.INT]) {
				c.ExpectErr = deferr
				m.EXPECT().A(a, *a, []byte{}).Return(test1.INT(0), deferr)
			})
			c.BREAK("Test 2: четное success", func(c *tshort.Case[test1.INT]) {
				c.ExpectOk = test1.INT(0)
				m.EXPECT().A(a, *a, []byte{}).Return(test1.INT(0), nil)
				m.EXPECT().B()
			})
		})
		c.CASE(func(c *tshort.Collector[test1.INT]) {
			*a = 3
			*a2 = *a - 1

			c.BREAK("Test 3: нечетное error", func(c *tshort.Case[test1.INT]) {
				c.ExpectErr = deferr
				m.EXPECT().A(a2, *a2, []byte{}).Return(test1.INT(0), deferr)
			})
			c.BREAK("Test 4: нечетное success", func(c *tshort.Case[test1.INT]) {
				c.ExpectOk = test1.INT(0)
				m.EXPECT().A(a2, *a2, []byte{}).Return(test1.INT(0), nil)
				m.EXPECT().B()
			})
		})
	}, func(t *testing.T, leaf *tshort.Case[test1.INT]) {
		b, err := Foo(a, m)
		assert.Equal(t, leaf.ExpectErr, err)
		assert.Equal(t, leaf.ExpectOk, b)
	})
}
