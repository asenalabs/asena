package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler

type Chain struct {
	middlewares []Middleware
}

func New(m ...Middleware) Chain {
	return Chain{middlewares: m}
}

func (c Chain) Then(h http.Handler) http.Handler {
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		h = c.middlewares[i](h)
	}
	return h
}
