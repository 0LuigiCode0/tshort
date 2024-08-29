package tshort

import (
	"fmt"
	"runtime"
	"testing"

	tutils "github.com/0LuigiCode0/tshort/internal/utils"
)

type TShort struct {
	stages map[string]*stage
	init   func(t *testing.T)
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

func Init(init func(t *testing.T)) *TShort {
	return &TShort{
		init:   init,
		stages: map[string]*stage{},
		cases:  []*_case{},
	}
}

// Добавляет новый стейдж
//   - name - имя нового стейджа, если начинается с '@', то это имя пропускается при наименовании кейса
//   - f - логика стейджа
//   - next - набор последующих стейджей
func (ts *TShort) AddStage(name string, f func(), next ...string) *TShort {
	ts.stages[name] = &stage{f, next, true}
	return ts
}

// Запускает тесты на каждый кейс
//
//	вызывает t.Run()
func (ts *TShort) Run(t *testing.T, f func(t *testing.T)) {
	ts.scan()

	for _, _case := range ts.cases {
		t.Run(_case.name, func(t *testing.T) {
			ts.init(t)

			for _, stage := range _case.stages {
				stage()
			}

			rec(t, func() { f(t) })
		})
	}
}

// Спасает от паник, возвращая трейс и ошибку на каждое падение
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

// Создает кейсы из цепочек стейджей, основываясь на из связях
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

// Ищет корневые стейджи, далее от них пойдет построение цепочек
func (ts *TShort) findRoot(next []string) {
	for _, s := range next {
		stage, ok := ts.stages[s]
		if !ok {
			panic("stage " + s + " not found")
		}
		stage.root = false

		ts.findRoot(stage.next)
	}
}

// Непосредственно стоит цепочки
//
//	если name начинается с '@', то это имя пропускается при наименовании кейса
func (ts *TShort) buildPipelines(name string, stage *stage, pipelines []func()) {
	if len(name) > 0 && name[0] == '@' {
		name = ""
	}
	if len(stage.next) > 0 {
		for _, nextName := range stage.next {
			stage = ts.stages[nextName]

			newpipe := make([]func(), len(pipelines))
			copy(newpipe, pipelines)
			newpipe = append(newpipe, stage.f)

			if len(nextName) > 0 && nextName[0] == '@' {
				ts.buildPipelines(name, stage, newpipe)
			} else {
				ts.buildPipelines(tutils.Join("->", name, nextName), stage, newpipe)
			}
		}
	} else {
		ts.cases = append(ts.cases, &_case{name, pipelines})
	}
}
