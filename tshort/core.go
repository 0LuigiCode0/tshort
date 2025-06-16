package tshort

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	tutils "github.com/0LuigiCode0/tshort/utils"
)

type TShort struct {
	stages map[string]*stage
	init   func(t *testing.T)
	cases  []*_case
	root   []string
	sep    string
}

type stage struct {
	name string
	f    func()
	next []string
}

type _case struct {
	name   string
	stages []func()
}

func Init(init func(t *testing.T), sep string, rootStage ...string) *TShort {
	return &TShort{
		init:   init,
		stages: map[string]*stage{},
		cases:  []*_case{},
		root:   rootStage,
		sep:    sep,
	}
}

// Добавляет новый стейдж
//   - name - имя нового стейджа, если начинается с '@', то это имя пропускается при наименовании кейса
//   - f - логика стейджа
//   - next - набор последующих стейджей
func (ts *TShort) AddStage(name string, f func(), next ...string) *TShort {
	ts.stages[name] = &stage{name, f, next}
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
	for _, name := range ts.root {
		stage := ts.findRoot(name)
		ts.buildPipelines("", stage, []func(){stage.f})
	}
}

// Ищет корневые стейджи, далее от них пойдет построение цепочек
func (ts *TShort) findRoot(s string) *stage {
	stage, ok := ts.stages[s]
	if !ok {
		panic("stage " + s + " not found")
	}
	return stage
}

// Непосредственно стоит цепочки
//
//	если name начинается с '@', то это имя пропускается при наименовании кейса
func (ts *TShort) buildPipelines(name string, stage *stage, pipelines []func()) {
	names := strings.Split(stage.name, ts.sep)
	newNames := make([]string, 0, len(names))
	for _, v := range names {
		fmt.Print(v)
		if len(v) > 0 && v[0] != '@' {
			newNames = append(newNames, v)
		}
	}
	name = tutils.Join("->", name, tutils.Join(ts.sep, newNames...))

	if len(stage.next) > 0 {
		for _, nextName := range stage.next {
			stage, ok := ts.stages[nextName]
			if !ok {
				panic("stage " + nextName + " not found in case " + name)
			}

			newpipe := make([]func(), len(pipelines), len(pipelines)+1)
			copy(newpipe, pipelines)
			newpipe = append(newpipe, stage.f)

			ts.buildPipelines(name, stage, newpipe)
		}
	} else {
		ts.cases = append(ts.cases, &_case{name, pipelines})
	}
}
