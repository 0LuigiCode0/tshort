package tutils

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func Equal(t *testing.T, x, y []any) {
	if len(x) != len(y) {
		fmt.Println("\tFAIL: the number of variables being compared is not equal")
		t.FailNow()
	}
	for i := 0; i < len(x); i++ {
		v1 := reflect.ValueOf(x[i])
		v2 := reflect.ValueOf(y[i])
		if !v1.Equal(v2) {
			fmt.Printf("\tFAIL: x[%d] %v(%v) not equal y[%d] %v(%v)\n", i, v1.Type().String(), v1, i, v2.Type().String(), v2)
			t.Fail()
		}
	}
}

func Join[data ~string](sep string, ss ...data) string {
	l := 0
	lsep := len(sep)
	for i := 0; i < len(ss); i++ {
		if ss[i] == "" {
			ss = append(ss[:i], ss[i+1:]...)
			i--
			continue
		}
		l += len(ss[i]) + lsep
	}

	b := strings.Builder{}
	if len(ss) > 0 {
		b.Grow(l)
		b.WriteString(string(ss[0]))
	} else {
		return ""
	}

	for _, s := range ss[1:] {
		b.WriteString(string(sep))
		b.WriteString(string(s))
	}
	return b.String()
}

func JoinF[data any](sep string, f func(i int, s data) (string, bool), args ...data) string {
	l := 0
	lsep := len(sep)
	ss := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		if s, ok := f(i, args[i]); ok {
			l += len(s) + lsep
			ss = append(ss, s)
		}
	}

	b := strings.Builder{}
	if len(ss) > 0 {
		b.Grow(l)
		b.WriteString(ss[0])
	} else {
		return ""
	}

	for _, s := range ss[1:] {
		b.WriteString(sep)
		b.WriteString(s)
	}
	return b.String()
}

func Convert[in, out any](_in []in, f func(int, in) (out, bool)) (_out []out) {
	for i, in := range _in {
		v, ok := f(i, in)
		if ok {
			_out = append(_out, v)
		}
	}
	return
}

// TODO: удалить
func Print(v ...any) {
	format := ""
	for range v {
		format += "%#v\n"
	}
	fmt.Printf(format, v...)
}
