// Package capture is a middleware that captures requests/responses size, status and headers.
//
// For another middleware to get those attributes of a requests, this middleware
// should be added before in the middleware chain.
//
//     	handler, _ := NewHandler()
//     	chain := alice.New().
//     	     Append(WrapHandler(handler)).
//     	     Append(myOtherMiddleware).
//     	     then(...)
//
// As this middleware stores those data in the request's context, the data can
// be retrieved at anytime after the ServerHTTP.
//
//     func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request, next http.Handler) {
//     ...
//     	crw := capture.GetResponseWriter(req.Context())
//     	fmt.Println(crw.Size)
//     }
package capture

import (
	"context"
	"net/http"

	"github.com/containous/alice"
)

type key string

// Handler will store each request data to its context.
type Handler struct{}

// WrapHandler Wraps capture handler into an Alice Constructor.
func WrapHandler(handler *Handler) alice.Constructor {
	return func(next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			handler.ServeHTTP(rw, req, next)
		}), nil
	}
}

// NewHandler creates a new Handler.
func NewHandler() (*Handler, error) {
	return &Handler{}, nil
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request, next http.Handler) {
	ctx := req.Context()
	if req.Body != nil {
		rr := newRequestReader(req.Body)
		ctx = context.WithValue(ctx, capturedRRData, rr)
		req.Body = rr
	}

	crw := newResponseWriter(rw)
	ctx = context.WithValue(ctx, capturedRWData, crw)
	next.ServeHTTP(crw, req.WithContext(ctx))
}
