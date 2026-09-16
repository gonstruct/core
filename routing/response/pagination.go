package response

import "github.com/gonstruct/core/routing/resource"

type meta interface {
	Resource() resource.Resource
}

type offsetPaginationMeta struct {
	Total       int
	Count       int
	PerPage     int
	CurrentPage int
	LastPage    int
}

func (m offsetPaginationMeta) Resource() resource.Resource {
	return resource.Resource{
		"total":       m.Total,
		"count":       m.Count,
		"perPage":     m.PerPage,
		"currentPage": m.CurrentPage,
		"lastPage":    m.LastPage,
	}
}

type offsetPaginationQuery interface {
	PerPage(...int) int
	Offset() int
}

func makeOffsetPaginationMeta(total int64, count int, query offsetPaginationQuery) offsetPaginationMeta {
	limit := query.PerPage()
	page := 1
	if query.Offset() > 0 {
		page = (query.Offset() / limit) + 1
	}

	lastPage := 1
	if total > 0 {
		lastPage = int((total + int64(limit) - 1) / int64(limit))
	}

	return offsetPaginationMeta{
		Total:       int(total),
		Count:       int(count),
		PerPage:     limit,
		CurrentPage: page,
		LastPage:    lastPage,
	}
}
