package main

import (
	"testing"
)

type TShort struct {
	cases      []string
	pipelines  map[string][]_stage
	middleware middleware
}

type _stage func()
type middleware func(t *testing.T)

func NewTShort(middleware middleware, cases ...string) *TShort {
	ts := &TShort{
		cases:      cases,
		pipelines:  make(map[string][]_stage, len(cases)),
		middleware: middleware,
	}
	for _, _case := range cases {
		ts.pipelines[_case] = make([]_stage, 0, 8)
	}

	return ts
}

func (ts *TShort) Stage(f _stage, cases ...string) {
	for _, _case := range cases {
		if _, ok := ts.pipelines[_case]; ok {
			ts.pipelines[_case] = append(ts.pipelines[_case], f)
		} else {
			panic("an attempt to add a non-existent case")
		}
	}
}

func (ts *TShort) Run(t *testing.T, execute _stage) {
	for _, _case := range ts.cases {
		t.Run(_case, func(t *testing.T) {
			if ts.middleware != nil {
				ts.middleware(t)
			}
			pipeline := ts.pipelines[_case]
			for _, f := range pipeline {
				f()
			}
			execute()
		})
	}
}

// func Equal(a, b []any) bool {

// }
