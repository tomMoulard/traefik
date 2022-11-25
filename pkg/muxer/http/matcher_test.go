package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/middlewares/requestdecorator"
)

func TestClientIPMatcher(t *testing.T) {
	testCases := []struct {
		desc          string
		rule          string
		expected      map[string]bool
		expectedError bool
	}{
		{
			desc:          "invalid ClientIP matcher",
			rule:          "ClientIP(`1`)",
			expectedError: true,
		},
		{
			desc:          "invalid ClientIP matcher (no parameter)",
			rule:          "ClientIP()",
			expectedError: true,
		},
		{
			desc:          "invalid ClientIP matcher (empty parameter)",
			rule:          "ClientIP(``)",
			expectedError: true,
		},
		{
			desc:          "invalid ClientIP matcher (too many parameters)",
			rule:          "ClientIP(`127.0.0.1`, `192.168.1.0/24`)",
			expectedError: true,
		},
		{
			desc: "valid ClientIP matcher",
			rule: "ClientIP(`127.0.0.1`)",
			expected: map[string]bool{
				"127.0.0.1":   true,
				"192.168.1.1": false,
			},
		},
		{
			desc: "valid ClientIP matcher but invalid remote address",
			rule: "ClientIP(`127.0.0.1`)",
			expected: map[string]bool{
				"1": false,
			},
		},
		{
			desc: "valid ClientIP matcher using CIDR",
			rule: "ClientIP(`192.168.1.0/24`)",
			expected: map[string]bool{
				"192.168.1.1":   true,
				"192.168.1.100": true,
				"192.168.2.1":   false,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			muxer, err := NewMuxer()
			require.NoError(t, err)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			err = muxer.AddRoute(test.rule, 0, handler)
			if test.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			results := make(map[string]bool)
			for remoteAddr := range test.expected {
				req := httptest.NewRequest(http.MethodGet, "https://example.com", http.NoBody)
				req.RemoteAddr = remoteAddr

				results[remoteAddr] = muxer.Match(req) != nil
			}
			assert.Equal(t, test.expected, results)
		})
	}
}

func TestMethodMatcher(t *testing.T) {
	testCases := []struct {
		desc          string
		rule          string
		expected      map[string]bool
		expectedError bool
	}{
		{
			desc:          "invalid Method matcher (no parameter)",
			rule:          "Method()",
			expectedError: true,
		},
		{
			desc:          "invalid Method matcher (empty parameter)",
			rule:          "Method(``)",
			expectedError: true,
		},
		{
			desc:          "invalid Method matcher (too many parameters)",
			rule:          "Method(`GET`, `POST`)",
			expectedError: true,
		},
		{
			desc: "valid Method matcher",
			rule: "Method(`GET`)",
			expected: map[string]bool{
				http.MethodGet:  true,
				http.MethodPost: false,
			},
		},
		{
			desc: "valid Method matcher (lower case)",
			rule: "Method(`get`)",
			expected: map[string]bool{
				http.MethodGet:  true,
				http.MethodPost: false,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			muxer, err := NewMuxer()
			require.NoError(t, err)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			err = muxer.AddRoute(test.rule, 0, handler)
			if test.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			results := make(map[string]bool)
			for method := range test.expected {
				req := httptest.NewRequest(method, "https://example.com", http.NoBody)

				results[method] = muxer.Match(req) != nil
			}
			assert.Equal(t, test.expected, results)
		})
	}
}

func TestHostMatcher(t *testing.T) {
	testCases := []struct {
		desc          string
		rule          string
		expected      map[string]bool
		expectedError bool
	}{
		{
			desc:          "invalid Host matcher (no parameter)",
			rule:          "Host()",
			expectedError: true,
		},
		{
			desc:          "invalid Host matcher (empty parameter)",
			rule:          "Host(``)",
			expectedError: true,
		},
		{
			desc:          "invalid Host matcher (non-ASCII)",
			rule:          "Host(`🦭.com`)",
			expectedError: true,
		},
		{
			desc:          "invalid Host matcher (too many parameters)",
			rule:          "Host(`example.com`, `example.org`)",
			expectedError: true,
		},
		{
			desc: "valid Host matcher",
			rule: "Host(`example.com`)",
			expected: map[string]bool{
				"https://example.com":      true,
				"https://example.com/path": true,
				"https://example.org":      false,
				"https://example.org/path": false,
			},
		},
		{
			desc: "valid Host matcher - matcher ending with a dot",
			rule: "Host(`example.com.`)",
			expected: map[string]bool{
				"https://example.com":       true,
				"https://example.com/path":  true,
				"https://example.org":       false,
				"https://example.org/path":  false,
				"https://example.com.":      true,
				"https://example.com./path": true,
				"https://example.org.":      false,
				"https://example.org./path": false,
			},
		},
		{
			desc: "valid Host matcher - URL ending with a dot",
			rule: "Host(`example.com`)",
			expected: map[string]bool{
				"https://example.com.":      true,
				"https://example.com./path": true,
				"https://example.org.":      false,
				"https://example.org./path": false,
			},
		},
		{
			desc: "valid Host matcher - puny-coded emoji",
			rule: "Host(`xn--9t9h.com`)",
			expected: map[string]bool{
				"https://xn--9t9h.com":      true,
				"https://xn--9t9h.com/path": true,
				"https://example.com":       false,
				"https://example.com/path":  false,
				// The request's sender must use puny-code.
				"https://🦭.com": false,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			muxer, err := NewMuxer()
			require.NoError(t, err)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			err = muxer.AddRoute(test.rule, 0, handler)
			if test.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// RequestDecorator is necessary for the host rule
			reqHost := requestdecorator.New(nil)

			results := make(map[string]bool)
			for calledURL := range test.expected {
				w := httptest.NewRecorder()

				req := httptest.NewRequest(http.MethodGet, calledURL, http.NoBody)

				reqHost.ServeHTTP(w, req, func(_ http.ResponseWriter, req *http.Request) {
					results[calledURL] = muxer.Match(req) != nil
				})
			}
			assert.Equal(t, test.expected, results)
		})
	}
}

func TestHostRegexpMatcher(t *testing.T) {
	testCases := []struct {
		desc          string
		rule          string
		expected      map[string]bool
		expectedError bool
	}{
		{
			desc:          "invalid HostRegexp matcher (no parameter)",
			rule:          "HostRegexp()",
			expectedError: true,
		},
		{
			desc:          "invalid HostRegexp matcher (empty parameter)",
			rule:          "HostRegexp(``)",
			expectedError: true,
		},
		{
			desc:          "invalid HostRegexp matcher (non-ASCII)",
			rule:          "HostRegexp(`🦭.com`)",
			expectedError: true,
		},
		{
			desc:          "invalid HostRegexp matcher (invalid regexp)",
			rule:          "HostRegexp(`(example.com`)",
			expectedError: true,
		},
		{
			desc:          "invalid HostRegexp matcher (too many parameters)",
			rule:          "HostRegexp(`example.com`, `example.org`)",
			expectedError: true,
		},
		{
			desc: "valid HostRegexp matcher",
			rule: "HostRegexp(`^[a-zA-Z-]+\\.com$`)",
			expected: map[string]bool{
				"https://example.com":      true,
				"https://example.com/path": true,
				"https://example.org":      false,
				"https://example.org/path": false,
			},
		},
		{
			desc: "valid HostRegexp matcher with Traefik v2 syntax",
			rule: "HostRegexp(`{domain:[a-zA-Z-]+\\.com}`)",
			expected: map[string]bool{
				"https://example.com":      false,
				"https://example.com/path": false,
				"https://example.org":      false,
				"https://example.org/path": false,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			muxer, err := NewMuxer()
			require.NoError(t, err)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			err = muxer.AddRoute(test.rule, 0, handler)
			if test.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			results := make(map[string]bool)
			for calledURL := range test.expected {
				req := httptest.NewRequest(http.MethodGet, calledURL, http.NoBody)
				results[calledURL] = muxer.Match(req) != nil
			}
			assert.Equal(t, test.expected, results)
		})
	}
}

func TestPathMatcher(t *testing.T) {
	testCases := []struct {
		desc          string
		rule          string
		expected      map[string]bool
		expectedError bool
	}{
		{
			desc:          "invalid Path matcher (no parameter)",
			rule:          "Path()",
			expectedError: true,
		},
		{
			desc:          "invalid Path matcher (empty parameter)",
			rule:          "Path(``)",
			expectedError: true,
		},
		{
			desc:          "invalid Path matcher (no leading /)",
			rule:          "Path(`css`)",
			expectedError: true,
		},
		{
			desc:          "invalid Path matcher (too many parameters)",
			rule:          "Path(`/css`, `/js`)",
			expectedError: true,
		},
		{
			desc: "valid Path matcher",
			rule: "Path(`/css`)",
			expected: map[string]bool{
				"https://example.com":              false,
				"https://example.com/html":         false,
				"https://example.org/css":          true,
				"https://example.com/css":          true,
				"https://example.com/css/":         false,
				"https://example.com/css/main.css": false,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			muxer, err := NewMuxer()
			require.NoError(t, err)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			err = muxer.AddRoute(test.rule, 0, handler)
			if test.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			results := make(map[string]bool)
			for calledURL := range test.expected {
				req := httptest.NewRequest(http.MethodGet, calledURL, http.NoBody)
				results[calledURL] = muxer.Match(req) != nil
			}
			assert.Equal(t, test.expected, results)
		})
	}
}

func TestPathRegexpMatcher(t *testing.T) {
	testCases := []struct {
		desc          string
		rule          string
		expected      map[string]bool
		expectedError bool
	}{
		{
			desc:          "invalid PathRegexp matcher (no parameter)",
			rule:          "PathRegexp()",
			expectedError: true,
		},
		{
			desc:          "invalid PathRegexp matcher (empty parameter)",
			rule:          "PathRegexp(``)",
			expectedError: true,
		},
		{
			desc:          "invalid PathRegexp matcher (invalid regexp)",
			rule:          "PathRegexp(`/(css`)",
			expectedError: true,
		},
		{
			desc:          "invalid PathRegexp matcher (too many parameters)",
			rule:          "PathRegexp(`/css`, `/js`)",
			expectedError: true,
		},
		{
			desc: "valid PathRegexp matcher",
			rule: "PathRegexp(`^/(css|js)`)",
			expected: map[string]bool{
				"https://example.com":              false,
				"https://example.com/html":         false,
				"https://example.org/css":          true,
				"https://example.com/CSS":          false,
				"https://example.com/css":          true,
				"https://example.com/css/":         true,
				"https://example.com/css/main.css": true,
				"https://example.com/js":           true,
				"https://example.com/js/":          true,
				"https://example.com/js/main.js":   true,
			},
		},
		{
			desc: "valid PathRegexp matcher with Traefik v2 syntax",
			rule: `PathRegexp("/{path:(css|js)}")`,
			expected: map[string]bool{
				"https://example.com":                 false,
				"https://example.com/html":            false,
				"https://example.org/css":             false,
				"https://example.com/{path:css}":      true,
				"https://example.com/{path:css}/":     true,
				"https://example.com/%7Bpath:css%7D":  true,
				"https://example.com/%7Bpath:css%7D/": true,
				"https://example.com/{path:js}":       true,
				"https://example.com/{path:js}/":      true,
				"https://example.com/%7Bpath:js%7D":   true,
				"https://example.com/%7Bpath:js%7D/":  true,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			muxer, err := NewMuxer()
			require.NoError(t, err)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			err = muxer.AddRoute(test.rule, 0, handler)
			if test.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			results := make(map[string]bool)
			for calledURL := range test.expected {
				req := httptest.NewRequest(http.MethodGet, calledURL, http.NoBody)
				results[calledURL] = muxer.Match(req) != nil
			}
			assert.Equal(t, test.expected, results)
		})
	}
}

func TestPathPrefixMatcher(t *testing.T) {
	testCases := []struct {
		desc          string
		rule          string
		expected      map[string]bool
		expectedError bool
	}{
		{
			desc:          "invalid PathPrefix matcher (no parameter)",
			rule:          "PathPrefix()",
			expectedError: true,
		},
		{
			desc:          "invalid PathPrefix matcher (empty parameter)",
			rule:          "PathPrefix(``)",
			expectedError: true,
		},
		{
			desc:          "invalid PathPrefix matcher (no leading /)",
			rule:          "PathPrefix(`css`)",
			expectedError: true,
		},
		{
			desc:          "invalid PathPrefix matcher (too many parameters)",
			rule:          "PathPrefix(`/css`, `/js`)",
			expectedError: true,
		},
		{
			desc: "valid PathPrefix matcher",
			rule: `PathPrefix("/css")`,
			expected: map[string]bool{
				"https://example.com":              false,
				"https://example.com/html":         false,
				"https://example.org/css":          true,
				"https://example.com/css":          true,
				"https://example.com/css/":         true,
				"https://example.com/css/main.css": true,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			muxer, err := NewMuxer()
			require.NoError(t, err)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			err = muxer.AddRoute(test.rule, 0, handler)
			if test.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			results := make(map[string]bool)
			for calledURL := range test.expected {
				req := httptest.NewRequest(http.MethodGet, calledURL, http.NoBody)
				results[calledURL] = muxer.Match(req) != nil
			}
			assert.Equal(t, test.expected, results)
		})
	}
}

func TestHeaderMatcher(t *testing.T) {
	testCases := []struct {
		desc          string
		rule          string
		expected      map[*http.Header]bool
		expectedError bool
	}{
		{
			desc:          "invalid Header matcher (no parameter)",
			rule:          "Header()",
			expectedError: true,
		},
		{
			desc:          "invalid Header matcher (missing value parameter)",
			rule:          "Header(`X-Forwarded-Host`)",
			expectedError: true,
		},
		{
			desc:          "invalid Header matcher (missing value parameter)",
			rule:          "Header(`X-Forwarded-Host`, ``)",
			expectedError: true,
		},
		{
			desc:          "invalid Header matcher (missing key parameter)",
			rule:          "Header(``, `example.com`)",
			expectedError: true,
		},
		{
			desc:          "invalid Header matcher (too many parameters)",
			rule:          "Header(`X-Forwarded-Host`, `example.com`, `example.org`)",
			expectedError: true,
		},
		{
			desc: "valid Header matcher",
			rule: "Header(`X-Forwarded-Proto`, `https`)",
			expected: map[*http.Header]bool{
				{"X-Forwarded-Proto": []string{"https"}}:         true,
				{"x-forwarded-proto": []string{"https"}}:         false,
				{"x-forwarded-proto": []string{"HTTPS"}}:         false,
				{"X-Forwarded-Proto": []string{"http", "https"}}: true,
				{"X-Forwarded-Proto": []string{"https", "http"}}: true,
				{"X-Forwarded-Host": []string{"example.com"}}:    false,
			},
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			muxer, err := NewMuxer()
			require.NoError(t, err)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			err = muxer.AddRoute(test.rule, 0, handler)
			if test.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			for headers := range test.expected {
				req := httptest.NewRequest(http.MethodGet, "https://example.com", http.NoBody)
				req.Header = *headers
				assert.Equal(t, test.expected[headers], muxer.Match(req) != nil, *headers)
			}
		})
	}
}

func TestHeaderRegexpMatcher(t *testing.T) {
	testCases := []struct {
		desc          string
		rule          string
		expected      map[*http.Header]bool
		expectedError bool
	}{
		{
			desc:          "invalid HeaderRegexp matcher (no parameter)",
			rule:          "HeaderRegexp()",
			expectedError: true,
		},
		{
			desc:          "invalid HeaderRegexp matcher (missing value parameter)",
			rule:          "HeaderRegexp(`X-Forwarded-Host`)",
			expectedError: true,
		},
		{
			desc:          "invalid HeaderRegexp matcher (missing value parameter)",
			rule:          "HeaderRegexp(`X-Forwarded-Host`, ``)",
			expectedError: true,
		},
		{
			desc:          "invalid HeaderRegexp matcher (missing key parameter)",
			rule:          "HeaderRegexp(``, `example.com`)",
			expectedError: true,
		},
		{
			desc:          "invalid HeaderRegexp matcher (invalid regexp)",
			rule:          "HeaderRegexp(`X-Forwarded-Host`,`(example.com`)",
			expectedError: true,
		},
		{
			desc:          "invalid HeaderRegexp matcher (too many parameters)",
			rule:          "HeaderRegexp(`X-Forwarded-Host`, `example.com`, `example.org`)",
			expectedError: true,
		},
		{
			desc: "valid HeaderRegexp matcher",
			rule: "HeaderRegexp(`X-Forwarded-Proto`, `^https?$`)",
			expected: map[*http.Header]bool{
				{"X-Forwarded-Proto": []string{"http"}}:        true,
				{"x-forwarded-proto": []string{"http"}}:        false,
				{"x-forwarded-proto": []string{"HTTPS"}}:       false,
				{"X-Forwarded-Proto": []string{"https"}}:       true,
				{"X-Forwarded-Proto": []string{"HTTPS"}}:       false,
				{"X-Forwarded-Proto": []string{"ws", "https"}}: true,
				{"X-Forwarded-Host": []string{"example.com"}}:  false,
			},
		},
		{
			desc: "valid HeaderRegexp matcher with Traefik v2 syntax",
			rule: "HeaderRegexp(`X-Forwarded-Proto`, `http{secure:s?}`)",
			expected: map[*http.Header]bool{
				{"X-Forwarded-Proto": []string{"http"}}:                 false,
				{"X-Forwarded-Proto": []string{"https"}}:                false,
				{"x-forwarded-proto": []string{"HTTPS"}}:                false,
				{"X-Forwarded-Proto": []string{"http{secure:}"}}:        true,
				{"X-Forwarded-Proto": []string{"HTTP{secure:}"}}:        false,
				{"X-Forwarded-Proto": []string{"http{secure:s}"}}:       true,
				{"X-Forwarded-Proto": []string{"http{secure:S}"}}:       false,
				{"X-Forwarded-Proto": []string{"HTTPS"}}:                false,
				{"X-Forwarded-Proto": []string{"ws", "http{secure:s}"}}: true,
				{"X-Forwarded-Host": []string{"example.com"}}:           false,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			muxer, err := NewMuxer()
			require.NoError(t, err)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			err = muxer.AddRoute(test.rule, 0, handler)
			if test.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			for headers := range test.expected {
				req := httptest.NewRequest(http.MethodGet, "https://example.com", http.NoBody)
				req.Header = *headers
				assert.Equal(t, test.expected[headers], muxer.Match(req) != nil, *headers)
			}
		})
	}
}

func TestQueryMatcher(t *testing.T) {
	testCases := []struct {
		desc          string
		rule          string
		expected      map[string]bool
		expectedError bool
	}{
		{
			desc:          "invalid Query matcher (no parameter)",
			rule:          "Query()",
			expectedError: true,
		},
		{
			desc:          "invalid Query matcher (empty key, one parameter)",
			rule:          "Query(``)",
			expectedError: true,
		},
		{
			desc:          "invalid Query matcher (empty key)",
			rule:          "Query(``, `traefik`)",
			expectedError: true,
		},
		{
			desc:          "invalid Query matcher (empty value)",
			rule:          "Query(`q`, ``)",
			expectedError: true,
		},
		{
			desc:          "invalid Query matcher (too many parameters)",
			rule:          "Query(`q`, `traefik`, `proxy`)",
			expectedError: true,
		},
		{
			desc: "valid Query matcher",
			rule: "Query(`q`, `traefik`)",
			expected: map[string]bool{
				"https://example.com":                     false,
				"https://example.com?q=traefik":           true,
				"https://example.com?rel=ddg&q=traefik":   true,
				"https://example.com?q=traefik&q=proxy":   true,
				"https://example.com?q=awesome&q=traefik": true,
				"https://example.com?q=nginx":             false,
				"https://example.com?rel=ddg":             false,
				"https://example.com?q=TRAEFIK":           false,
				"https://example.com?Q=traefik":           false,
				"https://example.com?rel=traefik":         false,
			},
		},
		{
			desc: "valid Query matcher with empty value",
			rule: "Query(`mobile`)",
			expected: map[string]bool{
				"https://example.com":             false,
				"https://example.com?mobile":      true,
				"https://example.com?mobile=true": false,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			muxer, err := NewMuxer()
			require.NoError(t, err)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			err = muxer.AddRoute(test.rule, 0, handler)
			if test.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			results := make(map[string]bool)
			for calledURL := range test.expected {
				req := httptest.NewRequest(http.MethodGet, calledURL, http.NoBody)
				results[calledURL] = muxer.Match(req) != nil
			}
			assert.Equal(t, test.expected, results)
		})
	}
}

func TestQueryRegexpMatcher(t *testing.T) {
	testCases := []struct {
		desc          string
		rule          string
		expected      map[string]bool
		expectedError bool
	}{
		{
			desc:          "invalid QueryRegexp matcher (no parameter)",
			rule:          "QueryRegexp()",
			expectedError: true,
		},
		{
			desc:          "invalid QueryRegexp matcher (empty parameter)",
			rule:          "QueryRegexp(``)",
			expectedError: true,
		},
		{
			desc:          "invalid QueryRegexp matcher (invalid regexp)",
			rule:          "QueryRegexp(`q`, `(traefik`)",
			expectedError: true,
		},
		{
			desc:          "invalid QueryRegexp matcher (too many parameters)",
			rule:          "QueryRegexp(`q`, `traefik`, `proxy`)",
			expectedError: true,
		},
		{
			desc: "valid QueryRegexp matcher",
			rule: "QueryRegexp(`q`, `^(traefik|nginx)$`)",
			expected: map[string]bool{
				"https://example.com":                     false,
				"https://example.com?q=traefik":           true,
				"https://example.com?rel=ddg&q=traefik":   true,
				"https://example.com?q=traefik&q=proxy":   true,
				"https://example.com?q=awesome&q=traefik": true,
				"https://example.com?q=TRAEFIK":           false,
				"https://example.com?Q=traefik":           false,
				"https://example.com?rel=traefik":         false,
				"https://example.com?q=nginx":             true,
				"https://example.com?rel=ddg&q=nginx":     true,
				"https://example.com?q=nginx&q=proxy":     true,
				"https://example.com?q=awesome&q=nginx":   true,
				"https://example.com?q=NGINX":             false,
				"https://example.com?Q=nginx":             false,
				"https://example.com?rel=nginx":           false,
				"https://example.com?q=haproxy":           false,
				"https://example.com?rel=ddg":             false,
			},
		},
		{
			desc: "valid QueryRegexp matcher",
			rule: "QueryRegexp(`q`, `^.*$`)",
			expected: map[string]bool{
				"https://example.com":                     false,
				"https://example.com?q=traefik":           true,
				"https://example.com?rel=ddg&q=traefik":   true,
				"https://example.com?q=traefik&q=proxy":   true,
				"https://example.com?q=awesome&q=traefik": true,
				"https://example.com?q=TRAEFIK":           true,
				"https://example.com?Q=traefik":           false,
				"https://example.com?rel=traefik":         false,
				"https://example.com?q=nginx":             true,
				"https://example.com?rel=ddg&q=nginx":     true,
				"https://example.com?q=nginx&q=proxy":     true,
				"https://example.com?q=awesome&q=nginx":   true,
				"https://example.com?q=NGINX":             true,
				"https://example.com?Q=nginx":             false,
				"https://example.com?rel=nginx":           false,
				"https://example.com?q=haproxy":           true,
				"https://example.com?rel=ddg":             false,
			},
		},
		{
			desc: "valid QueryRegexp matcher with Traefik v2 syntax",
			rule: "QueryRegexp(`q`, `{value:(traefik|nginx)}`)",
			expected: map[string]bool{
				"https://example.com?q=traefik":         false,
				"https://example.com?q={value:traefik}": true,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			muxer, err := NewMuxer()
			require.NoError(t, err)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			err = muxer.AddRoute(test.rule, 0, handler)
			if test.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			results := make(map[string]bool)
			for calledURL := range test.expected {
				req := httptest.NewRequest(http.MethodGet, calledURL, http.NoBody)
				results[calledURL] = muxer.Match(req) != nil
			}
			assert.Equal(t, test.expected, results)
		})
	}
}
