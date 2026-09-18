package app_test

import (
	"sync"
	"testing"

	"github.com/gonstruct/core/app"
)

type service struct{ built int }

func bindApp(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "testing")

	app.New().SetInstance()
}

func TestSingletonBuildsOnceAndShares(t *testing.T) {
	bindApp(t)

	builds := 0
	app.Singleton(func() *service {
		builds++

		return &service{built: builds}
	})

	if builds != 0 {
		t.Fatal("built before anything asked for it")
	}

	first, second := app.Make[*service](), app.Make[*service]()

	if first != second || builds != 1 {
		t.Errorf("built %d times, same instance: %v", builds, first == second)
	}
}

func TestBindBuildsEveryTime(t *testing.T) {
	bindApp(t)

	builds := 0
	app.Bind(func() *service {
		builds++

		return &service{built: builds}
	})

	if first, second := app.Make[*service](), app.Make[*service](); first == second || builds != 2 {
		t.Errorf("built %d times, same instance: %v", builds, first == second)
	}
}

func TestProvideHandsInWhatExists(t *testing.T) {
	bindApp(t)

	given := &service{built: 42}
	app.Provide(given)

	if app.Make[*service]() != given {
		t.Error("not the provided instance")
	}

	// A later binding of the same type replaces it, which is what a test
	// swapping a fake in relies on.
	app.Provide(&service{built: 43})

	if app.Make[*service]().built != 43 {
		t.Error("the later binding did not win")
	}
}

func TestForgetBuildsAgain(t *testing.T) {
	bindApp(t)

	builds := 0
	app.Singleton(func() *service {
		builds++

		return &service{built: builds}
	})

	before := app.Make[*service]()
	app.Forget[*service]()
	after := app.Make[*service]()

	if before == after || builds != 2 {
		t.Errorf("built %d times, same instance: %v", builds, before == after)
	}

	// Forgetting what was never bound is nothing.
	app.Forget[int]()
}

func TestMakePanicsWhenNothingIsBound(t *testing.T) {
	bindApp(t)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Error("no panic")
		}
	}()

	app.Make[string]()
}

func TestTypesDoNotCollide(t *testing.T) {
	bindApp(t)

	type other struct{}

	app.Singleton(func() *service { return &service{built: 1} })
	app.Singleton(func() *other { return &other{} })

	if app.Make[*service]().built != 1 {
		t.Error("the other type's binding was resolved")
	}
}

func TestSingletonUnderConcurrency(t *testing.T) {
	bindApp(t)

	var (
		mutex  sync.Mutex
		builds int
		wait   sync.WaitGroup
	)

	app.Singleton(func() *service {
		mutex.Lock()
		defer mutex.Unlock()

		builds++

		return &service{}
	})

	for range 50 {
		wait.Go(func() { app.Make[*service]() })
	}

	wait.Wait()

	if builds != 1 {
		t.Errorf("built %d times", builds)
	}
}
