package factory

import (
	"context"

	"github.com/aarondl/sqlboiler/v4/boil"
)

type Model interface {
	Reload(ctx context.Context, exec boil.ContextExecutor) error
	ReloadG(ctx context.Context) error
	Insert(context.Context, boil.ContextExecutor, boil.Columns) error
	InsertG(context.Context, boil.Columns) error
}

type factory[T Model] interface {
	Definition() T
}

type Factory[F factory[M], M Model] struct {
	context context.Context
	quietly bool
	hooks   hookOptions[M]
	states  []func(M)
	factory F
}

func NewFactory[F factory[M], M Model](factory F) *Factory[F, M] {
	f := &Factory[F, M]{
		context: context.Background(),
		factory: factory,
	}

	if configurable, ok := any(factory).(interface{ Configure(*Factory[F, M]) }); ok {
		configurable.Configure(f)
	}

	if hookable, ok := any(factory).(interface{ AfterMaking(M) }); ok {
		f.hooks.factoryAfterMaking = append(f.hooks.factoryAfterMaking, hookable.AfterMaking)
	}

	if hookable, ok := any(factory).(interface{ AfterCreating(M) }); ok {
		f.hooks.factoryAfterCreate = append(f.hooks.factoryAfterCreate, hookable.AfterCreating)
	}

	return f
}

func (f *Factory[F, M]) clone() *Factory[F, M] {
	clone := &Factory[F, M]{
		context: f.context,
		quietly: f.quietly,
		factory: f.factory,
	}

	clone.hooks.afterMaking = append([]func(M){}, f.hooks.afterMaking...)
	clone.hooks.afterCreating = append([]func(M){}, f.hooks.afterCreating...)
	clone.hooks.factoryAfterMaking = append([]func(M){}, f.hooks.factoryAfterMaking...)
	clone.hooks.factoryAfterCreate = append([]func(M){}, f.hooks.factoryAfterCreate...)
	clone.states = append([]func(M){}, f.states...)

	return clone
}

func (f *Factory[F, M]) Quietly() *Factory[F, M] {
	clone := f.clone()
	clone.quietly = true
	return clone
}

func (f *Factory[F, M]) State(fn func(M)) *Factory[F, M] {
	clone := f.clone()
	clone.states = append(clone.states, fn)
	return clone
}

func (f *Factory[F, M]) build() M {
	definition := f.factory.Definition()

	for _, state := range f.states {
		state(definition)
	}

	for _, hook := range f.hooks.afterMaking {
		hook(definition)
	}

	for _, hook := range f.hooks.factoryAfterMaking {
		hook(definition)
	}

	return definition
}

func (f *Factory[F, M]) persist(definition M) {
	context := f.context
	if f.quietly {
		context = boil.SkipHooks(context) // This is something i don't want but no other option for now (referring sqlboiler)
	}

	if err := definition.InsertG(context, boil.Infer()); err != nil {
		panic("failed to create model: " + err.Error())
	}

	for _, hook := range f.hooks.afterCreating {
		hook(definition)
	}

	for _, hook := range f.hooks.factoryAfterCreate {
		hook(definition)
	}
}

func (f *Factory[F, M]) Refresh(model M) {
	if err := model.ReloadG(f.context); err != nil {
		panic("failed to refresh model: " + err.Error())
	}
}
