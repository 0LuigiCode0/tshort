package internal

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
		if !reflect.DeepEqual(x[i], y[i]) {
			fmt.Printf("\tFAIL: x[%d] %v not equal y[%d] %v\n", i, x[i], i, y[i])
			t.Fail()
		}
	}
}

func Join[data ~string](sep data, ss ...data) string {
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
	b.Grow(l)

	if len(ss) > 0 {
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
