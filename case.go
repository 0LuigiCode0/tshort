package tshort

import (
	"strconv"
	"testing"
)

type Case interface {
	CASE(caseF CaseFunc)
	BREAK(name string, caseF CaseFunc)
}

type CaseFunc func(t *testing.T, c Case)

type _case struct {
	id   string
	t    *testing.T
	next []*_case
	f    CaseFunc
}

func (c *_case) CASE(caseF CaseFunc) {
	next := &_case{
		id: c.id + "." + strconv.Itoa(len(c.next)),
		t:  c.t,
		f:  caseF,
	}
	caseF(c.t, next)
	c.next = append(c.next, next)
}

func (c *_case) BREAK(_ string, caseF CaseFunc) {}
