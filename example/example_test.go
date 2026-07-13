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
	// объявление моковых классов и класса тестирования
	m := mocks.NewMockDoo(gomock.NewController(t))
	// объявление входящих данных и ожидаемых
	a := new(int)
	a2 := new(int)
	var wantB test1.INT
	var wantErr error

	tshort.Run(t, func(t *testing.T, c tshort.Case) {
		wantB = 0
		wantErr = nil

		c.CASE(func(t *testing.T, c tshort.Case) {
			*a = 4

			c.BREAK("Test 1: четное error", func(t *testing.T, c tshort.Case) {
				wantErr = deferr
				m.EXPECT().A(a, *a, []byte{}).Return(0, deferr)
			})
			c.BREAK("Test 2: четное success", func(t *testing.T, c tshort.Case) {
				m.EXPECT().A(a, *a, []byte{}).Return(0, nil)
				m.EXPECT().B()
			})
		})
		c.CASE(func(t *testing.T, c tshort.Case) {
			*a = 3
			*a2 = *a - 1

			c.BREAK("Test 3: нечетное error", func(t *testing.T, c tshort.Case) {
				wantErr = deferr
				m.EXPECT().A(a2, *a2, []byte{}).Return(0, deferr)
			})
			c.BREAK("Test 4: нечетное success", func(t *testing.T, c tshort.Case) {
				m.EXPECT().A(a2, *a2, []byte{}).Return(0, nil)
				m.EXPECT().B()
			})
		})
	}, func(t *testing.T) {
		b, err := Foo(a, m)
		assert.Equal(t, []any{b, err}, []any{wantB, wantErr})
	})
}
