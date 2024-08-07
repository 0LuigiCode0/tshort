package tshort

import "testing"

type TestStruct struct {
	A string
	B *TestStruct
	C []byte
	D map[any]*TestStruct
}

func BenchmarkCopy(b *testing.B) {
	t := &TestStruct{A: "hello", B: &TestStruct{A: "world", C: []byte{1}}, C: []byte{2}, D: map[any]*TestStruct{"Here": {A: "OPs"}}}
	for i := 0; i < b.N; i++ {
		Copy(t)
	}
}

// BenchmarkCopy-24
//  1299672	       935.2 ns/op	     784 B/op	      21 allocs/op
