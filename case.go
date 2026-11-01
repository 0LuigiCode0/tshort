package tshort

import "testing"

// Case — ожидаемый результат листа (BREAK): простая структура с двумя полями.
// BREAK конечная — подветок у неё нет и методов не нужно.
// T — тип ожидаемого значения при успехе.
type Case[T any] struct {
	ExpectOk  T
	ExpectErr error
}

// Collector — контекст выполнения CASE/BREAK. Держит стек фреймов:
// тело корня и тела всех CASE, по которым ведёт путь к листу, в порядке
// выполнения. CASE рекурсирует (добавляет своё тело в стек и сразу его
// выполняет), BREAK сразу назначает подтест (перезапуск стека).
//
// T — контекст текущего прохода: на первом — переданный в Run t без
// Cleanup (см. Run), на перезапуске цепочки — подтест листа. Для gomock
// достаточно: gomock.NewController(c.T).
type Collector[T any] struct {
	T      TestContext           // контекст текущего прохода
	stack  []func(*Collector[T]) // цепочка: корень → ... → этот CASE
	replay bool                  // перезапуск цепочки: CASE/BREAK ничего не делают
	testF  func(*testing.T, *Case[T])
}

// Collecting — true на первом проходе, когда CASE рекурсирует и BREAK
// назначает подтесты. На этом проходе тестовые данные не должны быть
// «живыми»: например, gomock-контроллер на c.T не авто-Finish'ится.
func (c *Collector[T]) Collecting() bool {
	return !c.replay
}

// CASE — подветка: добавляет тело в стек и сразу выполняет его (рекурсия:
// CASE/BREAK внутри тела строят поддерево). При перезапуске цепочки
// (replay) ничего не делает.
func (c *Collector[T]) CASE(caseF func(*Collector[T])) {
	if c.replay {
		return
	}
	c.T.Helper()
	stack := make([]func(*Collector[T]), len(c.stack)+1)
	copy(stack, c.stack)
	stack[len(c.stack)] = caseF
	caseF(&Collector[T]{T: c.T, stack: stack, testF: c.testF})
}

// BREAK — лист: конкретный сценарий. Сразу назначает подтест: перезапуск
// всего стека (корень → ... → родительский CASE), затем тело BREAK
// (получает *Case[T] листа) и testF.
func (c *Collector[T]) BREAK(name string, caseF func(*Case[T])) {
	if c.replay {
		return
	}
	c.T.Helper()
	leaf := &Case[T]{}
	c.T.Run(name, func(t *testing.T) {
		for _, f := range c.stack {
			f(&Collector[T]{T: t, replay: true})
		}
		caseF(leaf)
		c.testF(t, leaf)
	})
}
