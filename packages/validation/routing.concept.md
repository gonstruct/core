type invoke[T any] interface {
	Invoke(request T)
}

func Invoke[C invoke[T], T any]() {
	var controller C
	var request T

	controller.Invoke(request)
}

type MyController struct{}

func (c MyController) Invoke(request MyRequest) {}

type MyRequest struct {
	Field string `json:"field" validate:"required"`
	Value int    `json:"value" validate:"required"`
}

func main() {
	Invoke[MyController]()
}