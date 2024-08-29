package tgen

import tutils "github.com/0LuigiCode0/tshort/internal/utils"

type iunit interface {
	String() string
}

func fParamString(i int, s *_unitParam) (string, bool) { return s.String(), true }
func fParamValue(i int, s *_unitParam) (string, bool)  { return s.Value(), true }

type _unitEmpty struct{}

func (u *_unitEmpty) String() string {
	return ""
}

func (u *_unitEmpty) StringShort() string {
	return ""
}

var unitEmpty = &_unitEmpty{}

// -------------------------------------------------------------------------- //
// MARK:unit
// -------------------------------------------------------------------------- //

type _unit struct {
	exp   iunit
	value string
}

func unit(value string) *_unit {
	return &_unit{
		value: value,
	}
}

func (u *_unit) String() string {
	return u.value
}

// -------------------------------------------------------------------------- //
// MARK:unitPtr
// -------------------------------------------------------------------------- //

type _unitPtr struct {
	_unit
}

func unitPtr(exp iunit) *_unitPtr {
	return &_unitPtr{
		_unit: _unit{exp: exp},
	}
}

func (u *_unitPtr) String() string {
	return "*" + u.exp.String()
}

// -------------------------------------------------------------------------- //
// MARK:unitParam
// -------------------------------------------------------------------------- //

type _unitParam struct {
	_unit
}

func unitParam(value string, exp iunit) *_unitParam {
	return &_unitParam{
		_unit: _unit{exp: exp, value: value},
	}
}

func (u *_unitParam) String() string {
	return tutils.Join(" ", u.value, u.exp.String())
}

func (u *_unitParam) Value() string {
	return u.value
}

func (u *_unitParam) Exp() string {
	return u.exp.String()
}

// -------------------------------------------------------------------------- //
// MARK:unitComp
// -------------------------------------------------------------------------- //

type _unitComp struct {
	_unit
}

func unitComp(value string, exp string) *_unitComp {
	return &_unitComp{
		_unit: _unit{exp: unit(exp), value: value},
	}
}

func (u *_unitComp) String() string {
	return tutils.Join(".", u.exp.String(), u.value)
}

// -------------------------------------------------------------------------- //
// MARK:unitEllipsis
// -------------------------------------------------------------------------- //

type _unitEllipsis struct {
	_unit
}

func unitEllipsis(exp iunit) *_unitEllipsis {
	return &_unitEllipsis{
		_unit: _unit{exp: exp},
	}
}

func (u *_unitEllipsis) String() string {
	return "..." + u.exp.String()
}

// -------------------------------------------------------------------------- //
// MARK:unitChan
// -------------------------------------------------------------------------- //

type chanDirect int

const (
	chan_ chanDirect = iota
	chan_in
	chan_out
)

type _unitChan struct {
	_unit
	chanDirect chanDirect
}

func unitChan(chanDirect chanDirect, exp iunit) *_unitChan {
	return &_unitChan{
		_unit:      _unit{exp: exp},
		chanDirect: chanDirect,
	}
}

func (u *_unitChan) String() string {
	var s string
	switch u.chanDirect {
	case chan_in:
		s = "chan<-"
	case chan_out:
		s = "<-chan "
	default:
		s = "chan "
	}
	return s + u.exp.String()
}

// -------------------------------------------------------------------------- //
// MARK:unitArray
// -------------------------------------------------------------------------- //

type _unitArray struct {
	_unit
	len iunit
}

func unitArray(len, exp iunit) *_unitArray {
	return &_unitArray{
		_unit: _unit{exp: exp},
		len:   len,
	}
}

func (u *_unitArray) String() string {
	return "[" + u.len.String() + "]" + u.exp.String()
}

// -------------------------------------------------------------------------- //
// MARK:unitSlice
// -------------------------------------------------------------------------- //

type _unitSlice struct {
	_unit
}

func unitSlice(exp iunit) *_unitSlice {
	return &_unitSlice{
		_unit: _unit{exp: exp},
	}
}

func (u *_unitSlice) String() string {
	return "[]" + u.exp.String()
}

// -------------------------------------------------------------------------- //
// MARK:unitGen
// -------------------------------------------------------------------------- //

type _unitGen struct {
	_unit
	args []*_unitParam
}

func unitGen(exp iunit, args []*_unitParam) *_unitGen {
	return &_unitGen{
		_unit: _unit{exp: exp},
		args:  args,
	}
}

func (u *_unitGen) String() string {
	if len(u.args) > 0 {
		return u.exp.String() + "[" + tutils.JoinF(",", fParamString, u.args...) + "]"
	}
	return u.exp.String()
}

func (u *_unitGen) Value() string {
	if len(u.args) > 0 {
		return u.exp.String() + "[" + tutils.JoinF(",", fParamValue, u.args...) + "]"
	}
	return u.exp.String()
}

// -------------------------------------------------------------------------- //
// MARK:unitFunc
// -------------------------------------------------------------------------- //

type _unitFunc struct {
	_unit
	in  []*_unitParam
	out []*_unitParam
}

func unitFunc(in, out []*_unitParam) *_unitFunc {
	return &_unitFunc{
		_unit: _unit{},
		in:    in,
		out:   out,
	}
}

func (u *_unitFunc) String() string {
	s := "func(" + tutils.JoinF(",", fParamString, u.in...) + ")"

	if len(u.out) > 0 {
		s += "(" + tutils.JoinF(",", fParamString, u.out...) + ")"
	}
	return s
}

// -------------------------------------------------------------------------- //
// MARK:unitMap
// -------------------------------------------------------------------------- //

type _unitMap struct {
	_unit
	key   *_unitParam
	value *_unitParam
}

func unitMap(key, value *_unitParam) *_unitMap {
	return &_unitMap{
		_unit: _unit{},
		key:   key,
		value: value,
	}
}

func (u *_unitMap) String() string {
	return "map[" + u.key.String() + "]" + u.value.String()
}
