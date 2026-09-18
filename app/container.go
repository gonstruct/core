package app

import (
	"fmt"
	"sync"
)

// The container is Laravel's, narrowed to what Go needs: a binding is keyed
// by its type, so the call site is typed and there is no string to mistype.
//
// What the application shares is declared at boot, next to its routes and
// providers, and resolved anywhere:
//
//	app.New(app.WithSingletons(singletons.All))               // boot
//	app.Make[*sociable.Manager]().Driver("github")             // anywhere
//
// Singleton builds once, on the first Make, and shares; Bind builds on every
// Make. Provide hands in something already built, after boot, which is how a
// test puts a fake where the application expects the real thing. Forget
// drops what a singleton built, so a test that changed the configuration
// gets a fresh one on the next Make.

// Binding is a declaration for WithSingletons: what to build for a type.
type Binding struct {
	key       any
	construct func() any
	shared    bool
}

// Singleton declares a T that is built once, on the first Make, and shared
// after. Laravel's singleton(). It is lazy so the declaration can precede the
// things the constructor needs.
func Singleton[T any](construct func() T) Binding {
	return Binding{key: key[T](), construct: func() any { return construct() }, shared: true}
}

// Bind declares a T that is built on every Make. Laravel's bind().
func Bind[T any](construct func() T) Binding {
	return Binding{key: key[T](), construct: func() any { return construct() }}
}

// Singletons is the list an application declares, for WithSingletons.
func Singletons(bindings ...Binding) []Binding {
	return bindings
}

// Provide binds a T that already exists, on the bound application. Laravel's
// instance(). A later Provide for the same type replaces the earlier one.
func Provide[T any](value T) {
	Instance().bind(Binding{key: key[T](), construct: func() any { return value }, shared: true})
}

// Make resolves a T from the bound application. Nothing bound for that type
// is a wiring mistake rather than a runtime condition, so it panics.
func Make[T any]() T {
	value, ok := Instance().resolve(key[T]())
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

// key is the type's identity without reflection: a typed nil pointer is a
// comparable value whose dynamic type is *T.
func key[T any]() any { return (*T)(nil) }

// resolver is a binding as the application holds it: the declaration, and
// what it built when it is shared.
type resolver struct {
	Binding

	once     sync.Once
	instance any
}

func (self *resolver) resolve() any {
	if !self.shared {
		return self.construct()
	}

	self.once.Do(func() { self.instance = self.construct() })

	return self.instance
}

func (self *App) bind(binding Binding) {
	self.bindings.Store(binding.key, &resolver{Binding: binding})
}

func (self *App) resolve(key any) (any, bool) {
	bound, ok := self.bindings.Load(key)
	if !ok {
		return nil, false
	}

	return bound.(*resolver).resolve(), true
}

// forget drops what a shared binding built. The declaration is kept, so the
// next Make builds again.
func (self *App) forget(key any) {
	bound, ok := self.bindings.Load(key)
	if !ok {
		return
	}

	self.bind(bound.(*resolver).Binding)
}
