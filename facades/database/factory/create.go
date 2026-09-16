package factory

func (f *Factory[F, M]) Make() M {
	return f.build()
}

func (f *Factory[F, M]) MakeMany(count int) []M {
	if count < 1 {
		return []M{}
	}

	models := make([]M, count)
	for index := 0; index < count; index++ {
		models[index] = f.Make()
	}

	return models
}

func (f *Factory[F, M]) Create() M {
	definition := f.build()
	f.persist(definition)
	return definition
}

func (f *Factory[F, M]) CreateQuietly() M {
	return f.Quietly().Create()
}

func (f *Factory[F, M]) CreateMany(count int) []M {
	if count < 1 {
		return []M{}
	}

	models := make([]M, count)
	for index := 0; index < count; index++ {
		models[index] = f.Create()
	}

	return models
}

func (f *Factory[F, M]) CreateManyQuietly(count int) []M {
	return f.Quietly().CreateMany(count)
}
