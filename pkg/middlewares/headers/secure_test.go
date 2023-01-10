package headers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
)

// Middleware tests based on https://github.com/unrolled/secure

func Test_newSecure_modifyResponse(t *testing.T) {
	testCases := []struct {
		desc     string
		cfg      dynamic.SecureHeaders
		expected http.Header
	}{
		{
			desc: "PermissionsPolicy",
			cfg: dynamic.SecureHeaders{
				PermissionsPolicy: "microphone=(),",
			},
			expected: http.Header{"Permissions-Policy": []string{"microphone=(),"}},
		},
		{
			desc: "STSSeconds",
			cfg: dynamic.SecureHeaders{
				STSSeconds:     1,
				ForceSTSHeader: true,
			},
			expected: http.Header{"Strict-Transport-Security": []string{"max-age=1"}},
		},
		{
			desc: "STSSeconds and STSPreload",
			cfg: dynamic.SecureHeaders{
				STSSeconds:     1,
				ForceSTSHeader: true,
				STSPreload:     true,
			},
			expected: http.Header{"Strict-Transport-Security": []string{"max-age=1; preload"}},
		},
		{
			desc: "CustomFrameOptionsValue",
			cfg: dynamic.SecureHeaders{
				CustomFrameOptionsValue: "foo",
			},
			expected: http.Header{"X-Frame-Options": []string{"foo"}},
		},
		{
			desc: "FrameDeny",
			cfg: dynamic.SecureHeaders{
				FrameDeny: true,
			},
			expected: http.Header{"X-Frame-Options": []string{"DENY"}},
		},
		{
			desc: "ContentTypeNosniff",
			cfg: dynamic.SecureHeaders{
				ContentTypeNosniff: true,
			},
			expected: http.Header{"X-Content-Type-Options": []string{"nosniff"}},
		},
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			secure, err := NewSecureHeader(context.Background(), emptyHandler, test.cfg, "mymiddleware")
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "/foo", nil)

			rw := httptest.NewRecorder()

			secure.ServeHTTP(rw, req)

			assert.Equal(t, test.expected, rw.Result().Header)
		})
	}
}

func TestNew_allowedHosts(t *testing.T) {
	testCases := []struct {
		desc     string
		fromHost string
		expected int
	}{
		{
			desc:     "Should accept the request when given a host that is in the list",
			fromHost: "foo.com",
			expected: http.StatusOK,
		},
		{
			desc:     "Should refuse the request when no host is given",
			fromHost: "",
			expected: http.StatusInternalServerError,
		},
		{
			desc:     "Should refuse the request when no matching host is given",
			fromHost: "boo.com",
			expected: http.StatusInternalServerError,
		},
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	cfg := dynamic.SecureHeaders{
		AllowedHosts: []string{"foo.com", "bar.com"},
	}

	mid, err := NewSecureHeader(context.Background(), emptyHandler, cfg, "foo")
	require.NoError(t, err)

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/foo", nil)
			req.Host = test.fromHost

			rw := httptest.NewRecorder()

			mid.ServeHTTP(rw, req)

			assert.Equal(t, test.expected, rw.Code)
		})
	}
}
