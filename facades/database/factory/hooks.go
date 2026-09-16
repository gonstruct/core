package factory

type hookOptions[M Model] struct {
	afterMaking        []func(M)
	afterCreating      []func(M)
	factoryAfterMaking []func(M)
	factoryAfterCreate []func(M)
}

func (f *Factory[T, M]) AfterMaking(fn func(M)) *Factory[T, M] {
	clone := f.clone()
	clone.hooks.afterMaking = append(clone.hooks.afterMaking, fn)
	return clone
}

func (f *Factory[T, M]) AfterCreating(fn func(M)) *Factory[T, M] {
	clone := f.clone()
	clone.hooks.afterCreating = append(clone.hooks.afterCreating, fn)
	return clone
}
