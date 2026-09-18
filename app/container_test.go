package app_test

import (
	"sync"
	"testing"

	"github.com/gonstruct/core/app"
)

type service struct{ built int }

// boot builds and binds an application with the declarations, the way an
// entrypoint does.
func boot(t *testing.T, bindings ...app.Binding) {
	t.Helper()
	t.Setenv("APP_ENV", "testing")

	app.New(app.WithSingletons(app.Singletons(bindings...))).SetInstance()
}

func TestSingletonBuildsOnceAndShares(t *testing.T) {
	builds := 0

	boot(t, app.Singleton(func() *service {
		builds++

		return &service{built: builds}
	}))

	if builds != 0 {
		t.Fatal("built before anything asked for it")
	}

	first, second := app.Make[*service](), app.Make[*service]()

	if first != second || builds != 1 {
		t.Errorf("built %d times, same instance: %v", builds, first == second)
	}
}

func TestBindBuildsEveryTime(t *testing.T) {
	builds := 0

	boot(t, app.Bind(func() *service {
		builds++

		return &service{built: builds}
	}))

	if first, second := app.Make[*service](), app.Make[*service](); first == second || builds != 2 {
		t.Errorf("built %d times, same instance: %v", builds, first == second)
	}
}

func TestProvideReplacesTheDeclaration(t *testing.T) {
	boot(t, app.Singleton(func() *service { return &service{built: 1} }))

	given := &service{built: 42}
	app.Provide(given)

	if app.Make[*service]() != given {
		t.Error("not the provided instance")
	}

	app.Provide(&service{built: 43})

	if app.Make[*service]().built != 43 {
		t.Error("the later Provide did not win")
	}
}

func TestForgetBuildsAgain(t *testing.T) {
	builds := 0

	boot(t, app.Singleton(func() *service {
		builds++

		return &service{built: builds}
	}))

	before := app.Make[*service]()
	app.Forget[*service]()
	after := app.Make[*service]()

	if before == after || builds != 2 {
		t.Errorf("built %d times, same instance: %v", builds, before == after)
	}

	// Forgetting what was never declared is nothing.
	app.Forget[int]()
}

func TestMakePanicsWhenNothingIsBound(t *testing.T) {
	boot(t)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Error("no panic")
		}
	}()

	app.Make[string]()
}

func TestTypesDoNotCollide(t *testing.T) {
	type other struct{}

	boot(t,
		app.Singleton(func() *service { return &service{built: 1} }),
		app.Singleton(func() *other { return &other{} }),
	)

	if app.Make[*service]().built != 1 {
		t.Error("the other type's binding was resolved")
	}
}

func TestApplicationsDoNotShareBindings(t *testing.T) {
	boot(t, app.Singleton(func() *service { return &service{built: 1} }))

	// Another application, bound afterwards, declared nothing.
	boot(t)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Error("the first application's binding leaked into the second")
		}
	}()

	app.Make[*service]()
}

func TestSingletonUnderConcurrency(t *testing.T) {
	var (
		mutex  sync.Mutex
		builds int
		wait   sync.WaitGroup
	)

	boot(t, app.Singleton(func() *service {
		mutex.Lock()
		defer mutex.Unlock()

		builds++

		return &service{}
	}))

	for range 50 {
		wait.Go(func() { app.Make[*service]() })
	}

	wait.Wait()

	if builds != 1 {
		t.Errorf("built %d times", builds)
	}
}
