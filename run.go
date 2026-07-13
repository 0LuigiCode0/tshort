package tshort

import "testing"

func Run(t *testing.T, caseF CaseFunc, testF func(t *testing.T)) {
	for_, test := tests
	t.Run("", func(t *testing.T) {
		testF(t)
	})
}
