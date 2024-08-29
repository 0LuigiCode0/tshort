package example

import (
	"errors"
	"testing"

	examplemock "github.com/0LuigiCode0/tshort/example/mocks"
	"github.com/0LuigiCode0/tshort/internal/tshort"
	tutils "github.com/0LuigiCode0/tshort/internal/utils"
)

var deferr = errors.New("hello")

// Получаем такой вывод в консоль
//
// === RUN   TestFoo
// === RUN   TestFoo/четное->a1.error
// === RUN   TestFoo/четное->success
// === RUN   TestFoo/нечетное->a2.error
// === RUN   TestFoo/нечетное->success
// --- PASS: TestFoo (0.00s)
//     --- PASS: TestFoo/четное->a1.error (0.00s)
//     --- PASS: TestFoo/четное->success (0.00s)
//     --- PASS: TestFoo/нечетное->a2.error (0.00s)
//     --- PASS: TestFoo/нечетное->success (0.00s)
// PASS

func TestFoo(t *testing.T) {
	// объявление моковых классов и класса тестирования
	m := examplemock.NewDoo(t)
	// объявление входящих данных и ожидаемых
	a := new(int)
	a2 := new(int)
	var wantB int
	var wantErr error

	ts := tshort.Init(func(t *testing.T) {
		wantB = 0
		wantErr = nil
		m.Interceptor(t)
	})
	// разбиваем проверяемы код на блоки и записывает их связывая с последующими, тем самым создавая цепочки вызовов
	ts.AddStage("четное", func() {
		*a = 4
	}, "a1.error", "@a1.success")
	{
		ts.AddStage("a1.error", func() {
			wantErr = deferr
			m.EXPECT().A(a, *a, []byte{}).Return(0, deferr)
		})

		ts.AddStage("@a1.success", func() {
			m.EXPECT().A(a, *a, []byte{}).Return(0, nil)
		}, "success")
	}

	ts.AddStage("нечетное", func() {
		*a = 3
		*a2 = *a - 1
	}, "a2.error", "@a2.success")
	{
		ts.AddStage("a2.error", func() {
			wantErr = deferr
			m.EXPECT().A(a2, *a2, []byte{}).Return(0, deferr)
		})

		ts.AddStage("@a2.success", func() {
			m.EXPECT().A(a2, *a2, []byte{}).Return(0, nil)
		}, "success")
	}

	ts.AddStage("success", func() {
		m.EXPECT().B()
	})

	ts.Run(t, func(t *testing.T) {
		b, err := Foo(a, m)
		tutils.Equal(t, []any{b, err}, []any{wantB, wantErr})
	})
}
