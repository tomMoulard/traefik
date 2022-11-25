package http

import (
	"bufio"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/middlewares/requestdecorator"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
)

func TestMuxer(t *testing.T) {
	testCases := []struct {
		desc          string
		rule          string
		headers       map[string]string
		remoteAddr    string
		expected      map[string]bool
		expectedError bool
	}{
		{
			desc:          "no tree",
			expectedError: true,
		},
		{
			desc:          "Rule with no matcher",
			rule:          "rulewithnotmatcher",
			expectedError: true,
		},
		{
			desc:          "Rule without quote",
			rule:          "Host(example.com)",
			expectedError: true,
		},
		{
			desc: "Host and PathPrefix",
			rule: "Host(`localhost`) && PathPrefix(`/css`)",
			expected: map[string]bool{
				"https://localhost/css": true,
				"https://localhost/js":  false,
			},
		},
		{
			desc: "Rule with Host OR Host",
			rule: "Host(`example.com`) || Host(`example.org`)",
			expected: map[string]bool{
				"https://example.com/css": true,
				"https://example.org/js":  true,
				"https://example.eu/html": false,
			},
		},
		{
			desc: "Rule with host OR (host AND path)",
			rule: `Host("example.com") || (Host("example.org") && Path("/css"))`,
			expected: map[string]bool{
				"https://example.com/css": true,
				"https://example.com/js":  true,
				"https://example.org/css": true,
				"https://example.org/js":  false,
				"https://example.eu/css":  false,
			},
		},
		{
			desc: "Rule with host OR host AND path",
			rule: `Host("example.com") || Host("example.org") && Path("/css")`,
			expected: map[string]bool{
				"https://example.com/css": true,
				"https://example.com/js":  true,
				"https://example.org/css": true,
				"https://example.org/js":  false,
				"https://example.eu/css":  false,
			},
		},
		{
			desc: "Rule with (host OR host) AND path",
			rule: `(Host("example.com") || Host("example.org")) && Path("/css")`,
			expected: map[string]bool{
				"https://example.com/css": true,
				"https://example.com/js":  false,
				"https://example.org/css": true,
				"https://example.org/js":  false,
				"https://example.eu/css":  false,
			},
		},
		{
			desc: "Rule with (host AND path) OR (host AND path)",
			rule: `(Host("example.com") && Path("/js")) || ((Host("example.org")) && Path("/css"))`,
			expected: map[string]bool{
				"https://example.com/css": false,
				"https://example.com/js":  true,
				"https://example.org/css": true,
				"https://example.org/js":  false,
				"https://example.eu/css":  false,
			},
		},
		{
			desc: "Rule case UPPER",
			rule: `PATHPREFIX("/css")`,
			expected: map[string]bool{
				"https://example.com/css": true,
				"https://example.com/js":  false,
			},
		},
		{
			desc: "Rule case lower",
			rule: `pathprefix("/css")`,
			expected: map[string]bool{
				"https://example.com/css": true,
				"https://example.com/js":  false,
			},
		},
		{
			desc: "Rule case CamelCase",
			rule: `PathPrefix("/css")`,
			expected: map[string]bool{
				"https://example.com/css": true,
				"https://example.com/js":  false,
			},
		},
		{
			desc: "Rule case Title",
			rule: `Pathprefix("/css")`,
			expected: map[string]bool{
				"https://example.com/css": true,
				"https://example.com/js":  false,
			},
		},
		{
			desc: "Rule with not",
			rule: `!Host("example.com")`,
			expected: map[string]bool{
				"https://example.org": true,
				"https://example.com": false,
			},
		},
		{
			desc: "Rule with not on multiple route with or",
			rule: `!(Host("example.com") || Host("example.org"))`,
			expected: map[string]bool{
				"https://example.eu/js":   true,
				"https://example.com/css": false,
				"https://example.org/js":  false,
			},
		},
		{
			desc: "Rule with not on multiple route with and",
			rule: `!(Host("example.com") && Path("/css"))`,
			expected: map[string]bool{
				"https://example.com/js":  true,
				"https://example.eu/css":  true,
				"https://example.com/css": false,
			},
		},
		{
			desc: "Rule with not on multiple route with and another not",
			rule: `!(Host("example.com") && !Path("/css"))`,
			expected: map[string]bool{
				"https://example.com/css": true,
				"https://example.org/css": true,
				"https://example.com/js":  false,
			},
		},
		{
			desc: "Rule with not on two rule",
			rule: `!Host("example.com") || !Path("/css")`,
			expected: map[string]bool{
				"https://example.com/js":  true,
				"https://example.org/css": true,
				"https://example.com/css": false,
			},
		},
		{
			desc: "Rule case with double not",
			rule: `!(!(Host("example.com") && Pathprefix("/css")))`,
			expected: map[string]bool{
				"https://example.com/css": true,
				"https://example.com/js":  false,
				"https://example.org/css": false,
			},
		},
		{
			desc: "Rule case with not domain",
			rule: `!Host("example.com") && Pathprefix("/css")`,
			expected: map[string]bool{
				"https://example.org/css": true,
				"https://example.org/js":  false,
				"https://example.com/css": false,
				"https://example.com/js":  false,
			},
		},
		{
			desc: "Rule with multiple host AND multiple path AND not",
			rule: `!(Host("example.com") && Path("/js"))`,
			expected: map[string]bool{
				"https://example.com/js":    false,
				"https://example.com/html":  true,
				"https://example.org/js":    true,
				"https://example.com/css":   true,
				"https://example.org/css":   true,
				"https://example.org/html":  true,
				"https://example.eu/images": true,
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

				req := testhelpers.MustNewRequest(http.MethodGet, calledURL, http.NoBody)

				// Useful for the ClientIP matcher
				req.RemoteAddr = test.remoteAddr

				for key, value := range test.headers {
					req.Header.Set(key, value)
				}

				reqHost.ServeHTTP(w, req, func(_ http.ResponseWriter, req *http.Request) {
					results[calledURL] = muxer.Match(req) != nil
					assert.Equal(t, results[calledURL], muxer.Match(req) != nil)
				})
			}

			assert.Equal(t, test.expected, results)
		})
	}
}

func Test_addRoutePriority(t *testing.T) {
	type Case struct {
		xFrom    string
		rule     string
		priority int
	}

	testCases := []struct {
		desc     string
		path     string
		cases    []Case
		expected string
	}{
		{
			desc: "Higher priority on second rule",
			path: "/my",
			cases: []Case{
				{
					xFrom:    "header1",
					rule:     "PathPrefix(`/my`)",
					priority: 10,
				},
				{
					xFrom:    "header2",
					rule:     "PathPrefix(`/my`)",
					priority: 20,
				},
			},
			expected: "header2",
		},
		{
			desc: "Higher priority on first rule",
			path: "/my",
			cases: []Case{
				{
					xFrom:    "header1",
					rule:     "PathPrefix(`/my`)",
					priority: 20,
				},
				{
					xFrom:    "header2",
					rule:     "PathPrefix(`/my`)",
					priority: 10,
				},
			},
			expected: "header1",
		},
		{
			desc: "Higher priority on second rule with different rule",
			path: "/mypath",
			cases: []Case{
				{
					xFrom:    "header1",
					rule:     "PathPrefix(`/mypath`)",
					priority: 10,
				},
				{
					xFrom:    "header2",
					rule:     "PathPrefix(`/my`)",
					priority: 20,
				},
			},
			expected: "header2",
		},
		{
			desc: "Higher priority on longest rule (longest first)",
			path: "/mypath",
			cases: []Case{
				{
					xFrom: "header1",
					rule:  "PathPrefix(`/mypath`)",
				},
				{
					xFrom: "header2",
					rule:  "PathPrefix(`/my`)",
				},
			},
			expected: "header1",
		},
		{
			desc: "Higher priority on longest rule (longest second)",
			path: "/mypath",
			cases: []Case{
				{
					xFrom: "header1",
					rule:  "PathPrefix(`/my`)",
				},
				{
					xFrom: "header2",
					rule:  "PathPrefix(`/mypath`)",
				},
			},
			expected: "header2",
		},
		{
			desc: "Higher priority on longest rule (longest third)",
			path: "/mypath",
			cases: []Case{
				{
					xFrom: "header1",
					rule:  "PathPrefix(`/my`)",
				},
				{
					xFrom: "header2",
					rule:  "PathPrefix(`/mypa`)",
				},
				{
					xFrom: "header3",
					rule:  "PathPrefix(`/mypath`)",
				},
			},
			expected: "header3",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			muxer, err := NewMuxer()
			require.NoError(t, err)

			for _, route := range test.cases {
				route := route
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-From", route.xFrom)
				})

				err := muxer.AddRoute(route.rule, route.priority, handler)
				require.NoError(t, err, route.rule)
			}

			w := httptest.NewRecorder()
			req := testhelpers.MustNewRequest(http.MethodGet, test.path, http.NoBody)

			handler := muxer.Match(req)
			require.NotNil(t, handler)

			handler.ServeHTTP(w, req)

			assert.Equal(t, test.expected, w.Header().Get("X-From"))
		})
	}
}

func TestParseDomains(t *testing.T) {
	testCases := []struct {
		description   string
		expression    string
		domain        []string
		errorExpected bool
	}{
		{
			description:   "Unknown rule",
			expression:    "Foobar(`foo.bar`,`test.bar`)",
			errorExpected: true,
		},
		{
			description: "No host rule",
			expression:  "Path(`/test`)",
		},
		{
			description: "Host rule and another rule",
			expression:  "Host(`foo.bar`) && Path(`/test`)",
			domain:      []string{"foo.bar"},
		},
		{
			description: "Host rule to trim and another rule",
			expression:  "Host(`Foo.Bar`) || Host(`bar.buz`) && Path(`/test`)",
			domain:      []string{"foo.bar", "bar.buz"},
		},
		{
			description: "Host rule to trim and another rule",
			expression:  "Host(`Foo.Bar`) && Path(`/test`)",
			domain:      []string{"foo.bar"},
		},
		{
			description: "Host rule with no domain",
			expression:  "Host() && Path(`/test`)",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.expression, func(t *testing.T) {
			t.Parallel()

			domains, err := ParseDomains(test.expression)

			if test.errorExpected {
				require.Errorf(t, err, "unable to parse correctly the domains in the Host rule from %q", test.expression)
			} else {
				require.NoError(t, err, "%s: Error while parsing domain.", test.expression)
			}

			assert.EqualValues(t, test.domain, domains, "%s: Error parsing domains from expression.", test.expression)
		})
	}
}

// TestEmptyHost is a non regression test for
// https://github.com/traefik/traefik/pull/9131
func TestEmptyHost(t *testing.T) {
	testCases := []struct {
		desc     string
		request  string
		rule     string
		expected bool
	}{
		{
			desc:     "HostRegexp with absolute-form URL with empty host with non-matching host header",
			request:  "GET http://@/ HTTP/1.1\r\nHost: example.com\r\n\r\n",
			rule:     "HostRegexp(`example.com`)",
			expected: true,
		},
		{
			desc:     "Host with absolute-form URL with empty host with non-matching host header",
			request:  "GET http://@/ HTTP/1.1\r\nHost: example.com\r\n\r\n",
			rule:     "Host(`example.com`)",
			expected: true,
		},
		{
			desc:     "HostRegexp with absolute-form URL with matching host header",
			request:  "GET http://example.com/ HTTP/1.1\r\nHost: example.org\r\n\r\n",
			rule:     "HostRegexp(`example.com`)",
			expected: true,
		},
		{
			desc:     "Host with absolute-form URL with matching host header",
			request:  "GET http://example.com/ HTTP/1.1\r\nHost: example.org\r\n\r\n",
			rule:     "Host(`example.com`)",
			expected: true,
		},
		{
			desc:     "HostRegexp with absolute-form URL with non-matching host header",
			request:  "GET http://example.com/ HTTP/1.1\r\nHost: example.org\r\n\r\n",
			rule:     "HostRegexp(`example.org`)",
			expected: false,
		},
		{
			desc:     "Host with absolute-form URL with non-matching host header",
			request:  "GET http://example.com/ HTTP/1.1\r\nHost: example.org\r\n\r\n",
			rule:     "Host(`example.org`)",
			expected: false,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			muxer, err := NewMuxer()
			require.NoError(t, err)

			err = muxer.AddRoute(test.rule, 0, handler)
			require.NoError(t, err)

			// RequestDecorator is necessary for the host rule
			reqHost := requestdecorator.New(nil)

			w := httptest.NewRecorder()

			req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader([]byte(test.request))))
			require.NoError(t, err)

			reqHost.ServeHTTP(w, req, func(_ http.ResponseWriter, req *http.Request) {
				assert.Equal(t, test.expected, muxer.Match(req) != nil)
			})
		})
	}
}
