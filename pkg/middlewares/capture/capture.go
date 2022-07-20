package capture

import (
	"context"
	"net/http"

	"github.com/containous/alice"
)

// Handler will store each request data to its context
type Handler struct {
}

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
		ctx = context.WithValue(ctx, CapturedRRData, rr)
		req.Body = rr
	}

	crw := newCaptureResponseWriter(rw)
	ctx = context.WithValue(ctx, CapturedRWData, crw)
	next.ServeHTTP(crw, req.WithContext(ctx))
}
