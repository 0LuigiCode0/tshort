package main

//go:generate tshort --intgen=Atest
type Boo[Data any] interface {
	A(*int, Data, []byte) (int, error)
	B()
}

// тестируемая функция
func Foo(a *int, boo Boo[int]) (b int, err error) {
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
