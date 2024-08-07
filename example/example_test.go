package example

import (
	"errors"
	"testing"

	mockexample "github.com/0LuigiCode0/tshort/example/mocks"
	. "github.com/0LuigiCode0/tshort/internal"
)

var deferr = errors.New("hello")

func TestFoo(t *testing.T) {
	// объявление моковых классов и класса тестирования
	m := mockexample.NewBoo[int](t)
	ts := Init()
	// объявление входящих данных и ожидаемых
	a := new(int)
	a2 := new(int)
	a3 := new(int)
	var wantB int
	var wantErr error

	ts.AddStage("четное", func() {
		*a = 4
		*a3 = *a
	}, "a1.error", "a1.success")
	{
		ts.AddStage("a1.error", func() {
			wantErr = deferr
			m.EXPECT().A(a, *a, []byte{}).Return(0, deferr)
		})

		ts.AddStage("a1.success", func() {
			m.EXPECT().A(a, *a, []byte{}).Return(0, nil)
			*a3 *= 5
		}, "success")
	}

	ts.AddStage("нечетное", func() {
		*a = 3
		*a2 = *a - 1
	}, "a2.error", "a2.success")
	{
		ts.AddStage("a2.error", func() {
			wantErr = deferr
			m.EXPECT().A(a2, *a2, []byte{}).Return(0, deferr)
		})

		ts.AddStage("a2.success", func() {
			m.EXPECT().A(a2, *a2, []byte{}).Return(0, nil)
			*a3 = *a2
		}, "success")
	}

	ts.AddStage("success", func() {
		m.EXPECT().A(a3, *a3, []byte{}).Return(0, nil)
	})

	ts.Run(t, func(t *testing.T) {
		b, err := Foo(a, m)
		Equal(t, []any{b, err}, []any{wantB, wantErr})

		m.Interceptor(t)
		wantB = 0
		wantErr = nil
	})
}
