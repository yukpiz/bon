package bon

import "net/http"

type Group struct {
	mux         *Mux
	prefix      string
	middlewares []Middleware
}

func (g *Group) Group(pattern string, middlewares ...Middleware) *Group {
	g.prefix += pattern
	g.middlewares = append(g.middlewares, middlewares...)
	return g
}

func (g *Group) Use(middlewares ...Middleware) {
	g.middlewares = append(g.middlewares, middlewares...)
}

func (g *Group) Get(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle("GET", pattern, handlerFunc, middlewares...)
}

func (g *Group) Post(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle("POST", pattern, handlerFunc, middlewares...)
}

func (g *Group) Put(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle("PUT", pattern, handlerFunc, middlewares...)
}

func (g *Group) Delete(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle("DELETE", pattern, handlerFunc, middlewares...)
}

func (g *Group) Head(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle("HEAD", pattern, handlerFunc, middlewares...)
}

func (g *Group) Options(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle("OPTIONS", pattern, handlerFunc, middlewares...)
}

func (g *Group) Patch(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle("PATCH", pattern, handlerFunc, middlewares...)
}

func (g *Group) Connect(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle("CONNECT", pattern, handlerFunc, middlewares...)
}

func (g *Group) Trace(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle("TRACE", pattern, handlerFunc, middlewares...)
}

func (g *Group) Handle(method, pattern string, handler http.Handler, middlewares ...Middleware) {
	g.mux.Handle(method, g.prefix+pattern, handler, middlewares...)
}

func (g *Group) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}