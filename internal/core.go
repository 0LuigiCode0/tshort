package internal

import (
	"fmt"
	"runtime"
	"testing"
)

type TShort struct {
	stages map[string]*stage
	cases  []*_case
}

type stage struct {
	f    func()
	next []string
	root bool
}

type _case struct {
	name   string
	stages []func()
}

func Init() *TShort {
	return &TShort{
		stages: map[string]*stage{},
		cases:  []*_case{},
	}
}

func (ts *TShort) AddStage(name string, f func(), next ...string) *TShort {
	ts.stages[name] = &stage{f, next, true}
	return ts
}

func (ts *TShort) Run(t *testing.T, f func(t *testing.T)) {
	ts.scan()

	for _, _case := range ts.cases {
		for _, stage := range _case.stages {
			stage()
		}

		t.Run(_case.name, func(t *testing.T) {
			rec(t, func() { f(t) })
		})
	}
}

func rec(t *testing.T, f func()) {
	defer func() {
		if err := recover(); err != nil {
			pc := make([]uintptr, 2)
			runtime.Callers(5, pc)
			frames := runtime.CallersFrames(pc)

			fmt.Println("\t--- Trace")
			for {
				frame, ok := frames.Next()
				fmt.Printf("\t\t%s %d\n", frame.File, frame.Line)
				if !ok {
					break
				}
			}
			fmt.Printf("\t--- ERROR\n\t\t%v\n", err)

			t.Fail()
		}
	}()

	f()
}

func (ts *TShort) scan() {
	for _, v := range ts.stages {
		ts.findRoot(v.next)
	}

	for name, stage := range ts.stages {
		if stage.root {
			ts.buildPipelines(name, stage, []func(){stage.f})
		}
	}
}

func (ts *TShort) findRoot(next []string) {
	for _, s := range next {
		stage := ts.stages[s]
		stage.root = false

		ts.findRoot(stage.next)
	}
}

func (ts *TShort) buildPipelines(name string, stage *stage, pipelines []func()) {
	if len(stage.next) > 0 {
		for _, nextName := range stage.next {
			stage = ts.stages[nextName]

			newpipe := make([]func(), len(pipelines))
			copy(newpipe, pipelines)
			newpipe = append(newpipe, stage.f)

			ts.buildPipelines(Join("->", name, nextName), stage, newpipe)
		}
	} else {
		ts.cases = append(ts.cases, &_case{name, pipelines})
	}
}
