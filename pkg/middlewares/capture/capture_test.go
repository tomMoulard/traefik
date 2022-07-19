package capture

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/containous/alice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapture(t *testing.T) {
	wrapMiddleware := func(next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			crw := GetCapturedResponseWriter(req.Context())

			_, err := fmt.Fprintf(rw, "%d,%d,", crw.Size(), crw.Status())
			require.NoError(t, err)

			next.ServeHTTP(rw, req)

			_, err = fmt.Fprintf(rw, ",%d,%d", crw.Size(), crw.Status())
			require.NoError(t, err)
		}), nil
	}

	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, err := rw.Write([]byte("toto"))
		require.NoError(t, err)
	})

	captureHandler, err := NewHandler()
	require.NotNil(t, captureHandler)
	require.NoError(t, err)

	wrapped := WrapHandler(captureHandler)

	chain := alice.New()
	chain = chain.Append(wrapped)
	chain = chain.Append(wrapMiddleware)
	handlers, err := chain.Then(handler)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "http://foo/", nil)

	handlers.ServeHTTP(recorder, request)
	// 8 = len("0,0,toto")
	assert.Equal(t, "0,0,toto,8,200", recorder.Body.String())
}
