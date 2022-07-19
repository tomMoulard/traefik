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
	crw := newCaptureResponseWriter(rw)
	reqCaptured := req.WithContext(context.WithValue(req.Context(), CapturedRWData, crw))
	next.ServeHTTP(crw, reqCaptured)
}

func GetCapturedResponseWriter(ctx context.Context) capturer {
	c, ok := ctx.Value(CapturedRWData).(capturer)
	if !ok {
		panic("WTF?")
	}

	return c
}
