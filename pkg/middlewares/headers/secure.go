package headers

import (
	"context"
	"net/http"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/middlewares"
	"github.com/unrolled/secure"
)

type secureHeader struct {
	next   http.Handler
	secure *secure.Secure
	cfg    dynamic.SecureHeaders
}

// newSecure constructs a new secure instance with supplied options.
func NewSecureHeader(ctx context.Context, next http.Handler, cfg dynamic.SecureHeaders, name string) (http.Handler, error) {
	logger := middlewares.GetLogger(ctx, name, "SecureHeader")
	logger.Debug().Interface("config", cfg).Msg("Setting up SecureHeaders")

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

	return &secureHeader{
		next:   next,
		secure: secure.New(opt),
		cfg:    cfg,
	}, nil
}

func (s secureHeader) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	s.secure.HandlerFuncWithNextForRequestOnly(rw, req, func(writer http.ResponseWriter, request *http.Request) {
		s.next.ServeHTTP(newResponseModifier(writer, request, s.secure.ModifyResponseHeaders), request)
	})
}
