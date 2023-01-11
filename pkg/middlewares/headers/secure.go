package headers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/middlewares"
	"github.com/traefik/traefik/v2/pkg/middlewares/connectionheader"
	"github.com/unrolled/secure"
)

type secureHeader struct {
	next               http.Handler
	secure             *secure.Secure
	cfg                dynamic.SecureHeaders
	hasCorsHeaders     bool
	allowOriginRegexes []*regexp.Regexp
}

// newSecure constructs a new secure instance with supplied options.
func NewSecureHeader(ctx context.Context, next http.Handler, cfg dynamic.SecureHeaders, name string) (http.Handler, error) {
	logger := middlewares.GetLogger(ctx, name, "SecureHeader")

	hasCorsHeaders := cfg.HasCorsHeadersDefined()

	logger.Debug().
		Interface("config", cfg).
		Bool("cors", hasCorsHeaders).
		Msg("Setting up SecureHeaders")

	opt := secure.Options{
		BrowserXssFilter:        cfg.BrowserXSSFilter,
		ContentTypeNosniff:      cfg.ContentTypeNosniff,
		ForceSTSHeader:          cfg.ForceSTSHeader,
		FrameDeny:               cfg.FrameDeny,
		IsDevelopment:           cfg.IsDevelopment,
		STSIncludeSubdomains:    cfg.STSIncludeSubdomains,
		STSPreload:              cfg.STSPreload,
		ContentSecurityPolicy:   cfg.ContentSecurityPolicy,
		CustomBrowserXssValue:   cfg.CustomBrowserXSSValue,
		CustomFrameOptionsValue: cfg.CustomFrameOptionsValue,
		PublicKey:               cfg.PublicKey,
		ReferrerPolicy:          cfg.ReferrerPolicy,
		AllowedHosts:            cfg.AllowedHosts,
		HostsProxyHeaders:       cfg.HostsProxyHeaders,
		SSLProxyHeaders:         cfg.SSLProxyHeaders,
		STSSeconds:              cfg.STSSeconds,
		PermissionsPolicy:       cfg.PermissionsPolicy,
		SecureContextKey:        name,
	}
	secureHandler := &secureHeader{
		next:   next,
		secure: secure.New(opt),
		cfg:    cfg,
	}

	if !hasCorsHeaders {
		return secureHandler, nil
	}

	logger.Debug().Msgf("Setting up Cors from %v", cfg)

	regexes := make([]*regexp.Regexp, len(cfg.AccessControlAllowOriginListRegex))
	for i, str := range cfg.AccessControlAllowOriginListRegex {
		reg, err := regexp.Compile(str)
		if err != nil {
			return nil, fmt.Errorf("error occurred during origin parsing: %w", err)
		}
		regexes[i] = reg
	}

	secureHandler.allowOriginRegexes = regexes
	secureHandler.hasCorsHeaders = true

	return connectionheader.Remover(secureHandler), nil
}

func (s *secureHeader) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Handle Cors headers and preflight if configured.
	if isPreflight := s.processCorsHeaders(rw, req); isPreflight {
		return
	}

	s.secure.HandlerFuncWithNextForRequestOnly(rw, req, func(writer http.ResponseWriter, request *http.Request) {
		s.next.ServeHTTP(newResponseModifier(writer, request, s.secure.ModifyResponseHeaders), request)
	})
}

// processCorsHeaders processes the incoming request,
// and returns if it is a preflight request.
// If not a preflight, it handles the preRequestModifyCorsResponseHeaders.
func (s *secureHeader) processCorsHeaders(rw http.ResponseWriter, req *http.Request) bool {
	if !s.hasCorsHeaders {
		return false
	}

	reqAcMethod := req.Header.Get("Access-Control-Request-Method")
	originHeader := req.Header.Get("Origin")

	if reqAcMethod != "" && originHeader != "" && req.Method == http.MethodOptions {
		// If the request is an OPTIONS request with an Access-Control-Request-Method header,
		// and Origin headers, then it is a CORS preflight request,
		// and we need to build a custom response: https://www.w3.org/TR/cors/#preflight-request
		if s.cfg.AccessControlAllowCredentials {
			rw.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		allowHeaders := strings.Join(s.cfg.AccessControlAllowHeaders, ",")
		if allowHeaders != "" {
			rw.Header().Set("Access-Control-Allow-Headers", allowHeaders)
		}

		allowMethods := strings.Join(s.cfg.AccessControlAllowMethods, ",")
		if allowMethods != "" {
			rw.Header().Set("Access-Control-Allow-Methods", allowMethods)
		}

		allowed, match := s.isOriginAllowed(originHeader)
		if allowed {
			rw.Header().Set("Access-Control-Allow-Origin", match)
		}

		rw.Header().Set("Access-Control-Max-Age", strconv.Itoa(int(s.cfg.AccessControlMaxAge)))
		return true
	}

	return false
}

func (s *secureHeader) isOriginAllowed(origin string) (bool, string) {
	for _, item := range s.cfg.AccessControlAllowOriginList {
		if item == "*" || item == origin {
			return true, item
		}
	}

	for _, regex := range s.allowOriginRegexes {
		if regex.MatchString(origin) {
			return true, origin
		}
	}

	return false, ""
}
