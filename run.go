package tshort

import "testing"

// TestContext — то, что tshort нужно от контекста теста: отчёт об ошибках,
// подтесты, helper. Метода Cleanup здесь НАСКОРОМ — именно его смотрит
// gomock.NewController, чтобы решить про авто-Finish.
type TestContext interface {
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Helper()
	Run(name string, f func(t *testing.T)) bool
}

// quietT — TestContext без Cleanup: gomock-контроллер, созданный на нём,
// не авто-Finish'ится. Для первого прохода (сбор дерева): EXPECT'ы из тел
// CASE регистрируются, но не срабатывают.
type quietT struct {
	t TestContext
}

func (q quietT) Errorf(format string, args ...any) { q.t.Errorf(format, args...) }

func (q quietT) Fatalf(format string, args ...any) { q.t.Fatalf(format, args...) }

func (q quietT) Helper() { q.t.Helper() }

func (q quietT) Run(name string, f func(t *testing.T)) bool { return q.t.Run(name, f) }

// Run — выполняет тело корня. CASE рекурсия строит стек цепочки
// (корень → ... → CASE); каждый BREAK сразу назначает подтест:
// перезапуск стека, тело BREAK, затем testF.
//
// t — контекст теста (можно кастомный). На первом проходе тела получают
// его «подменённый» — без Cleanup (см. quietT), чтобы gomock-контроллер
// на c.T не авто-Finish'ился; на перезапуске цепочки c.T — подтест листа.
//
// T — тип ожидаемого значения при успехе; выводится из подписи тел
// (или указывается явно: Run[*legacy.Result](...)).
//
// Тела CASE получают *Collector (при перезапуске цепочки её методы ничего
// не делают), тела BREAK — *Case[T] листа, которую testF получает как leaf.
func Run[T any](t TestContext, caseF func(c *Collector[T]), testF func(t *testing.T, leaf *Case[T])) {
	t.Helper()
	caseF(&Collector[T]{T: quietT{t}, stack: []func(*Collector[T]){caseF}, testF: testF})
}
