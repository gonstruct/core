package app

import (
	"fmt"
	"sync"
)

// The container is Laravel's, narrowed to what Go needs: a binding is keyed
// by its type, so the call site is typed and there is no string to mistype.
//
//	app.Singleton(func() *sociable.Manager { return sociable.New(...) })  // in a provider
//	app.Make[*sociable.Manager]().Driver("github")                          // anywhere
//
// Singleton builds once and shares, Bind builds on every Make, Provide hands
// in something already built. Forget drops what a singleton built, so a test
// that changed the configuration gets a fresh one on the next Make.

// key is the type's identity without reflection: a typed nil pointer is a
// comparable value whose dynamic type is *T.
func key[T any]() any { return (*T)(nil) }

type binding struct {
	construct func() any
	shared    bool

	once     sync.Once
	instance any
}

func (self *binding) resolve() any {
	if !self.shared {
		return self.construct()
	}

	self.once.Do(func() { self.instance = self.construct() })

	return self.instance
}

func (self *App) bind(key any, construct func() any, shared bool) {
	self.bindings.Store(key, &binding{construct: construct, shared: shared})
}

func (self *App) make(key any) (any, bool) {
	bound, ok := self.bindings.Load(key)
	if !ok {
		return nil, false
	}

	return bound.(*binding).resolve(), true
}

// forget drops what a shared binding built. The constructor is kept, so the
// next make builds again.
func (self *App) forget(key any) {
	bound, ok := self.bindings.Load(key)
	if !ok {
		return
	}

	previous := bound.(*binding)
	self.bindings.Store(key, &binding{construct: previous.construct, shared: previous.shared})
}

// Singleton binds a T that is built once, on the first Make, and shared
// after. Laravel's singleton(). A provider registers before the things the
// constructor needs exist, which is why it is lazy.
func Singleton[T any](construct func() T) {
	Instance().bind(key[T](), func() any { return construct() }, true)
}

// Bind binds a T that is built on every Make. Laravel's bind().
func Bind[T any](construct func() T) {
	Instance().bind(key[T](), func() any { return construct() }, false)
}

// Provide binds a T that already exists. Laravel's instance(). It is how a
// test puts a fake where the application expects the real thing.
func Provide[T any](value T) {
	Instance().bind(key[T](), func() any { return value }, true)
}

// Make resolves a T. Nothing bound for that type is a wiring mistake rather
// than a runtime condition, so it panics.
func Make[T any]() T {
	value, ok := Instance().make(key[T]())
	if !ok {
		var zero T

		panic(fmt.Sprintf("app: nothing is bound for %T", zero))
	}

	return value.(T)
}

// Forget drops what a singleton built, so the next Make builds it again.
// Laravel's forgetInstance(). For a test that changed the configuration a
// singleton read.
func Forget[T any]() {
	Instance().forget(key[T]())
}
