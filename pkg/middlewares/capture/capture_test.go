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

	request, err := http.NewRequest(http.MethodGet, "http://foo/", nil)

	recorder := httptest.NewRecorder()
	handlers.ServeHTTP(recorder, request)
	// 8 = len("0,0,toto")
	assert.Equal(t, "0,0,toto,8,200", recorder.Body.String())
}

// BenchmarkCapture
// $ go test -bench=. ./pkg/middlewares/capture/
// goos: linux
// goarch: amd64
// pkg: github.com/traefik/traefik/v2/pkg/middlewares/capture
// cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
// BenchmarkCapture/2k-12            283904              4028 ns/op         508.49 MB/s        5072 B/op         14 allocs/op
// BenchmarkCapture/20k-12           140854              8622 ns/op        2375.24 MB/s       41936 B/op         14 allocs/op
// BenchmarkCapture/100k-12           45736             26887 ns/op        3808.60 MB/s      213968 B/op         14 allocs/op
// BenchmarkCapture/2k_captured-12   278564              4765 ns/op         429.78 MB/s        5552 B/op         18 allocs/op
// BenchmarkCapture/20k_captured-12  112636              9887 ns/op        2071.38 MB/s       42416 B/op         18 allocs/op
// BenchmarkCapture/100k_captured-12  43767             30369 ns/op        3371.81 MB/s      214448 B/op         18 allocs/op
// PASS
func BenchmarkCapture(b *testing.B) {
	testCases := []struct {
		name    string
		size    int
		capture bool
	}{
		{
			name: "2k",
			size: 2048,
		},
		{
			name: "20k",
			size: 20480,
		},
		{
			name: "100k",
			size: 102400,
		},
		{
			name:    "2k captured",
			size:    2048,
			capture: true,
		},
		{
			name:    "20k captured",
			size:    20480,
			capture: true,
		},
		{
			name:    "100k captured",
			size:    102400,
			capture: true,
		},
	}

	for _, test := range testCases {
		b.Run(test.name, func(b *testing.B) {
			baseBody := generateBytes(test.size)

			next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				n, err := rw.Write(baseBody)
				require.Equal(b, test.size, n)
				require.NoError(b, err)
			})

			req, err := http.NewRequest(http.MethodGet, "http://foo/", nil)
			require.NoError(b, err)

			chain := alice.New()
			if test.capture {
				captureHandler, err := NewHandler()
				require.NotNil(b, captureHandler)
				require.NoError(b, err)

				captureWrapped := WrapHandler(captureHandler)
				chain = chain.Append(captureWrapped)
			}
			handlers, err := chain.Then(next)
			require.NoError(b, err)

			b.ReportAllocs()
			b.SetBytes(int64(test.size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				runBenchmark(b, test.size, req, handlers)
			}
		})
	}
}

func runBenchmark(b *testing.B, size int, req *http.Request, handler http.Handler) {
	b.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if code := recorder.Code; code != 200 {
		b.Fatalf("Expected 200 but got %d", code)
	}

	assert.Equal(b, size, len(recorder.Body.String()))
}

func generateBytes(length int) []byte {
	var value []byte
	for i := 0; i < length; i++ {
		value = append(value, 0x61+byte(i%26))
	}
	return value
}
