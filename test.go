package tshort

import "testing"

type test struct {
	t *testing.T

	count int
	next  *test
}

func (t *test) CASE(caseF CaseFunc) {
	if t.count > 0 {
		t.count--
		caseF(t.t, t.next)
	}
}

func (t *test) BREAK(_ string, caseF CaseFunc) { t.CASE(caseF) }
