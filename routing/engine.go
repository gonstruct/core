package routing

import (
	"fmt"

	"github.com/gonstruct/core/routing/response"

	"github.com/gin-gonic/gin"
)

type ResourceController interface {
	List(context *gin.Context) response.Response
	Show(context *gin.Context) response.Response
	Create(context *gin.Context) response.Response
	Update(context *gin.Context) response.Response
	Delete(context *gin.Context) response.Response
}

type Middleware interface {
	Handle(context *gin.Context) response.Response
}

type Engine struct {
	*gin.Engine

	middlewares map[string]response.HandlerChain
}

type MiddlewareEngine struct {
	Global response.HandlerChain
	Named  map[string]response.HandlerChain
}

func NewEngine() *Engine {
	return &Engine{
		Engine:      gin.New(),
		middlewares: make(map[string]response.HandlerChain),
	}
}

type RouterGroup struct {
	*gin.RouterGroup

	middlewares map[string]response.HandlerChain
}

func (e *Engine) Group(path string, f func(router *RouterGroup)) {
	f(&RouterGroup{e.Engine.Group(path), e.middlewares})
}

func (e *Engine) Resource(path string, parameter string, controller ResourceController) {
	e.GET(path, controller.List)
	e.GET(path+"/:"+parameter, controller.Show)
	e.POST(path, controller.Create)
	e.PUT(path+"/:"+parameter, controller.Update)
	e.DELETE(path+"/:"+parameter, controller.Delete)
}

func (rg *RouterGroup) Group(path string, f func(router *RouterGroup)) {
	f(&RouterGroup{rg.RouterGroup.Group(path), rg.middlewares})
}

func (rg *RouterGroup) Resource(path string, parameter string, controller ResourceController) {
	rg.GET(path, controller.List)
	rg.GET(path+"/:"+parameter, controller.Show)
	rg.POST(path, controller.Create)
	rg.PUT(path+"/:"+parameter, controller.Update)
	rg.DELETE(path+"/:"+parameter, controller.Delete)
}

// GET overwrites the default handlers to return a Response interface.
func (e Engine) GET(path string, handlers ...response.Handler) gin.IRoutes {
	return e.Engine.GET(path, response.HandleChain(handlers)...)
}

func (e Engine) HEAD(path string, handlers ...response.Handler) gin.IRoutes {
	return e.Engine.HEAD(path, response.HandleChain(handlers)...)
}

func (e Engine) POST(path string, handlers ...response.Handler) gin.IRoutes {
	return e.Engine.POST(path, response.HandleChain(handlers)...)
}

func (e Engine) PATCH(path string, handlers ...response.Handler) gin.IRoutes {
	return e.Engine.PATCH(path, response.HandleChain(handlers)...)
}

func (e Engine) PUT(path string, handlers ...response.Handler) gin.IRoutes {
	return e.Engine.PUT(path, response.HandleChain(handlers)...)
}

func (e Engine) DELETE(path string, handlers ...response.Handler) gin.IRoutes {
	return e.Engine.DELETE(path, response.HandleChain(handlers)...)
}

func (e Engine) ANY(path string, handlers ...response.Handler) gin.IRoutes {
	return e.Engine.Any(path, response.HandleChain(handlers)...)
}

func (e Engine) UseMiddleware(handler func(c *gin.Context) response.Response) gin.IRoutes {
	return e.Engine.Use(response.Handle(handler))
}

func (e Engine) Use(name string) gin.IRoutes {
	if handlers, ok := e.middlewares[name]; ok {
		return e.Engine.Use(response.HandleChain(handlers)...)
	}

	panic(fmt.Errorf("middleware with name '%s' not found", name))
}

func (e Engine) NoMethod(handler response.Handler) {
	e.Engine.NoMethod(response.Handle(handler))
}

func (e Engine) NoRoute(handler response.Handler) {
	e.Engine.NoRoute(response.Handle(handler))
}

// router.Middleware("api", "someother").Group(func ()  {

// 	})
// func (e Engine) Middleware(handler func(c *gin.Context, f func(rg *RouterGroup)) Response)

func (rg RouterGroup) GET(path string, handlers ...response.Handler) gin.IRoutes {
	return rg.RouterGroup.GET(path, response.HandleChain(handlers)...)
}

func (rg RouterGroup) HEAD(path string, handlers ...response.Handler) gin.IRoutes {
	return rg.RouterGroup.HEAD(path, response.HandleChain(handlers)...)
}

func (rg RouterGroup) POST(path string, handlers ...response.Handler) gin.IRoutes {
	return rg.RouterGroup.POST(path, response.HandleChain(handlers)...)
}

func (rg RouterGroup) PATCH(path string, handlers ...response.Handler) gin.IRoutes {
	return rg.RouterGroup.PATCH(path, response.HandleChain(handlers)...)
}

func (rg RouterGroup) PUT(path string, handlers ...response.Handler) gin.IRoutes {
	return rg.RouterGroup.PUT(path, response.HandleChain(handlers)...)
}

func (rg RouterGroup) DELETE(path string, handlers ...response.Handler) gin.IRoutes {
	return rg.RouterGroup.DELETE(path, response.HandleChain(handlers)...)
}

func (rg RouterGroup) ANY(path string, handlers ...response.Handler) gin.IRoutes {
	return rg.RouterGroup.Any(path, response.HandleChain(handlers)...)
}

func (rg RouterGroup) UseMiddleware(handler func(c *gin.Context) response.Response) gin.IRoutes {
	return rg.RouterGroup.Use(response.Handle(handler))
}

func (rg RouterGroup) Use(name string) gin.IRoutes {
	if handlers, ok := rg.middlewares[name]; ok {
		return rg.RouterGroup.Use(response.HandleChain(handlers)...)
	}

	panic(fmt.Errorf("middleware with name '%s' not found", name))
}
