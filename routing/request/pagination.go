package request

type OffsetPaginationRequestQuery struct {
	_maxLimit int

	PerPageSetter *int `form:"perPage" validate:"omitempty,min=1"`
	PageSetter    *int `form:"page"    validate:"omitempty,min=1"`
}

func (q *OffsetPaginationRequestQuery) MaxLimit() int {
	if q._maxLimit > 0 {
		return q._maxLimit
	}

	return 100
}

func (q *OffsetPaginationRequestQuery) Offset() int {
	if q.PageSetter == nil || *q.PageSetter <= 1 {
		return 0
	}

	return (*q.PageSetter - 1) * q.PerPage()
}

func (q *OffsetPaginationRequestQuery) PerPage(maxPerPage ...int) int {
	if len(maxPerPage) == 1 {
		q._maxLimit = maxPerPage[0]
	}

	if q.PerPageSetter == nil || *q.PerPageSetter <= 0 || *q.PerPageSetter > q.MaxLimit() {
		return q.MaxLimit()
	}

	return *q.PerPageSetter
}
