package internal

import (
	"log"
	"reflect"
)

type cycler map[uintptr]reflect.Value

func Copy[data any](v data) data {
	cycle := cycler{}
	value, err := copyraiter(cycle, reflect.ValueOf(v))
	if err != nil {
		log.Println(err)
		return *new(data)
	}
	return value.Interface().(data)
}

func copyraiter(cycle cycler, elem reflect.Value) (out reflect.Value, err error) {
	if !elem.IsZero() {
		switch elem.Kind() {
		case reflect.Pointer:
			return cPointer(cycle, elem)
		case reflect.Slice:
			return cSlice(cycle, elem)
		case reflect.Array:
			return cArr(cycle, elem)
		case reflect.Struct:
			return cStruct(cycle, elem)
		case reflect.Map:
			return cMap(cycle, elem)
		case reflect.Chan:
			return cChan(cycle, elem)
		default:
			return cValue(elem)
		}
	}
	return elem, nil
}

func cValue(elem reflect.Value) (out reflect.Value, _ error) {
	out = reflect.New(elem.Type()).Elem()
	out.Set(elem)
	return
}

func cPointer(cycle cycler, elem reflect.Value) (out reflect.Value, err error) {
	var ok bool
	if out, ok = cycle[elem.Pointer()]; ok {
		return
	}

	out = reflect.New(elem.Type().Elem())
	value, err := copyraiter(cycle, elem.Elem())
	if err != nil {
		return out, err
	}
	if out.Elem().CanSet() && value.IsValid() {
		out.Elem().Set(value)
	}
	cycle[elem.Pointer()] = out
	return
}

func cSlice(cycle cycler, elem reflect.Value) (out reflect.Value, err error) {
	var ok bool
	if out, ok = cycle[elem.Pointer()]; ok {
		return
	}

	out = reflect.MakeSlice(elem.Type(), elem.Len(), elem.Cap())
	err = _cIndexEach(cycle, elem, out)
	if err != nil {
		return
	}
	cycle[elem.Pointer()] = out
	return
}

func cArr(cycle cycler, elem reflect.Value) (out reflect.Value, err error) {
	var ok bool
	if out, ok = cycle[elem.Pointer()]; ok {
		return
	}

	out = reflect.New(elem.Type()).Elem()
	err = _cIndexEach(cycle, elem, out)
	if err != nil {
		return
	}
	cycle[elem.Pointer()] = out
	return
}

func _cIndexEach(cycle cycler, elem, out reflect.Value) error {
	for i := 0; i < elem.Len(); i++ {
		value, err := copyraiter(cycle, elem.Index(i))
		if err != nil {
			return err
		}
		if out.Index(i).CanSet() && value.IsValid() {
			out.Index(i).Set(value)
		}
	}
	return nil
}

func cStruct(cycle cycler, elem reflect.Value) (out reflect.Value, err error) {
	out = reflect.New(elem.Type()).Elem()
	for i := 0; i < elem.NumField(); i++ {
		value, err := copyraiter(cycle, elem.Field(i))
		if err != nil {
			return out, err
		}
		if out.Field(i).CanSet() && value.IsValid() {
			out.Field(i).Set(value)
		}
	}
	return
}

func cMap(cycle cycler, elem reflect.Value) (out reflect.Value, err error) {
	var ok bool
	if out, ok = cycle[elem.Pointer()]; ok {
		return
	}

	out = reflect.MakeMap(elem.Type())
	maps := elem.MapRange()
	for maps.Next() {
		value, err := copyraiter(cycle, maps.Value())
		if err != nil {
			return out, err
		}
		if value.IsValid() {
			out.SetMapIndex(maps.Key(), value)
		}
	}
	cycle[elem.Pointer()] = out
	return
}

func cChan(cycle cycler, elem reflect.Value) (out reflect.Value, err error) {
	var ok bool
	if out, ok = cycle[elem.Pointer()]; ok {
		return
	}

	out = reflect.MakeChan(elem.Type(), elem.Cap())
	buf := make([]reflect.Value, 0, elem.Len())
	for i := 0; i < elem.Len(); i++ {
		x, ok := elem.TryRecv()
		if ok {
			buf = append(buf, x)
		}
	}
	for _, v := range buf {
		elem.TrySend(v)
		out.TrySend(v)
	}
	cycle[elem.Pointer()] = out
	return
}
