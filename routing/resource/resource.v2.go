package resource

import "github.com/gin-gonic/gin"

type V2ResourceInterface interface {
	// NOTE: this should not hinge on gin, but for this core it's fine
	Attributes(request *gin.Context) Resource
}

type V2CollectionInterface interface {
	Attributes(request *gin.Context) []Resource
	Length() int
}

type CollectionV2[I any, T V2ResourceInterface] struct {
	Items     []I
	Converter func(I) T
}

func NewCollectionV2[I any, T V2ResourceInterface](items []I, converter func(I) T) CollectionV2[I, T] {
	return CollectionV2[I, T]{
		Items:     items,
		Converter: converter,
	}
}

func (c CollectionV2[I, T]) Attributes(request *gin.Context) []Resource {
	collection := make([]Resource, len(c.Items))
	for i, item := range c.Items {
		collection[i] = c.Converter(item).Attributes(request)
	}
	return collection
}

func (c CollectionV2[I, T]) Length() int {
	return len(c.Items)
}
