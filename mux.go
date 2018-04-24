package bon

import (
	"net/http"
	"sync"
)

const (
	nodeKindStatic nodeKind = iota
	nodeKindParam
	nodeKindCatchAll
)

type (
	Mux struct {
		tree        *node
		middlewares []Middleware
		pool        sync.Pool
		maxParam    int
		NotFound    http.HandlerFunc
	}

	nodeKind uint8

	node struct {
		kind          nodeKind
		parent        *node
		children      map[string]*node
		paramChild    *node
		catchAllChild *node
		middlewares   []Middleware
		handler       http.Handler
		paramKey      string
	}

	Middleware func(http.Handler) http.Handler
)

func newMux() *Mux {
	m := &Mux{
		NotFound: http.NotFound,
	}

	m.pool = sync.Pool{
		New: func() interface{} {
			return m.NewContext()
		},
	}

	m.tree = newNode()
	return m
}

func newNode() *node {
	return &node{
		children: make(map[string]*node),
	}
}

func (n *node) newChild(child *node, edge string) *node {
	if len(n.children) == 0 {
		n.children = make(map[string]*node)
	}

	child.parent = n
	n.children[edge] = child
	return child
}

func isStaticPattern(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == ':' || pattern[i] == '*' {
			return false
		}
	}

	return true
}

func compensatePattern(pattern string) string {
	if len(pattern) > 0 {
		if pattern[0] != '/' {
			return "/" + pattern
		}
	}

	return pattern
}

func (m *Mux) Group(pattern string, middlewares ...Middleware) *Group {
	return &Group{
		mux:         m,
		middlewares: append(m.middlewares, middlewares...),
		prefix:      compensatePattern(pattern),
	}
}

func (m *Mux) Route(middlewares ...Middleware) *Route {
	return &Route{
		mux:         m,
		middlewares: middlewares,
	}
}

func (m *Mux) Use(middlewares ...Middleware) {
	m.middlewares = append(m.middlewares, middlewares...)
}

func (m *Mux) Get(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodGet, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Post(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodPost, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Put(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodPut, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Delete(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodDelete, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Head(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodHead, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Options(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodOptions, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Patch(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodPatch, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Connect(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodConnect, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Trace(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodTrace, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) FileServer(pattern, dir string) {
	if !isStaticPattern(pattern) {
		panic("It is not a static pattern")
	}

	if pattern[len(pattern)-1] != '/' {
		pattern += "/"
	}

	m.Handle(http.MethodGet, pattern+"*", http.StripPrefix(pattern, http.FileServer(http.Dir(dir))))
}

func (m *Mux) Handle(method, pattern string, handler http.Handler, middlewares ...Middleware) {
	parent := m.tree.children[method]

	if parent == nil {
		parent = m.tree.newChild(newNode(), method)
	}

	pattern = compensatePattern(pattern)

	if isStaticPattern(pattern) {
		if _, ok := parent.children[pattern]; !ok {
			child := newNode()
			child.middlewares = middlewares
			child.handler = handler
			parent.newChild(child, pattern)
		}

		return
	}

	var si, ei, pi int

	// i = 0 is '/'
	for i := 1; i < len(pattern); i++ {
		si = i
		ei = i

		for ; i < len(pattern); i++ {
			if si < ei {
				if pattern[i] == ':' || pattern[i] == '*' {
					panic("Parameter are not first")
				}
			}

			if pattern[i] == '/' {
				break
			}

			ei++
		}

		edge := pattern[si:ei]
		kind := nodeKindStatic
		var paramKey string

		switch edge[0] {
		case ':':
			paramKey = edge[1:]
			edge = ":"
			kind = nodeKindParam
		case '*':
			edge = "*"
			kind = nodeKindCatchAll
		}

		child, exist := parent.children[edge]

		if !exist {
			child = newNode()
		}

		child.kind = kind

		if len(paramKey) > 0 {
			child.paramKey = paramKey
			pi++
		}

		if i >= len(pattern)-1 {
			child.middlewares = middlewares
			child.handler = handler
		}

		switch child.kind {
		case nodeKindParam:
			parent.paramChild = child
		case nodeKindCatchAll:
			parent.catchAllChild = child
		}

		if exist {
			parent = child
			continue
		}

		parent = parent.newChild(child, edge)
	}

	if pi > m.maxParam {
		m.maxParam = pi
	}
}

func (m *Mux) lookup(r *http.Request) (*node, *Context) {
	var parent, child, backtrack *node

	if parent = m.tree.children[r.Method]; parent == nil {
		return nil, nil
	}

	rPath := r.URL.Path

	//STATIC PATH
	if child = parent.children[rPath]; child != nil {
		return child, nil
	}

	var si, ei int
	var ctx *Context

	for i := 1; i < len(rPath); i++ {
		si = i
		ei = i

		for ; i < len(rPath); i++ {
			if rPath[i] == '/' {
				break
			}

			ei++
		}

		edge := rPath[si:ei]

		if child = parent.children[edge]; child == nil {
			if child = parent.paramChild; child != nil {
				if ctx == nil {
					ctx = m.pool.Get().(*Context)
				}

				ctx.PutParam(child.paramKey, edge)
			} else {
				child = parent.catchAllChild
			}
		}

		if child != nil {
			if i >= len(rPath)-1 && child.handler != nil {
				return child, ctx
			}

			if parent.catchAllChild != nil && parent.catchAllChild.handler != nil {
				backtrack = parent.catchAllChild
			}

			if len(child.children) == 0 {
				if child.kind == nodeKindCatchAll && child.handler != nil {
					return child, ctx
				}

				break
			}

			parent = child
			continue
		}

		break
	}

	return backtrack, ctx
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if n, ctx := m.lookup(r); n != nil {
		if ctx != nil {
			r = ctx.WithContext(r)
		}

		if len(n.middlewares) == 0 {
			n.handler.ServeHTTP(w, r)

			if ctx != nil {
				m.pool.Put(ctx.reset())
			}

			return
		}

		h := n.middlewares[len(n.middlewares)-1](n.handler)

		for i := len(n.middlewares) - 2; i >= 0; i-- {
			h = n.middlewares[i](h)
		}

		h.ServeHTTP(w, r)

		if ctx != nil {
			m.pool.Put(ctx.reset())
		}

		return
	}

	m.NotFound.ServeHTTP(w, r)
}
