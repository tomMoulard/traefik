---
title: "Traefik Secure Headers Documentation"
description: "In Traefik Proxy, the HTTP secure headers middleware manages the security-related headers. Read the technical documentation."
---

# Secure Headers

Managing Secure Headers
{: .subtitle }

![Headers](../../assets/img/middleware/headers.png)

The Secure Headers middleware manages the security-related headers.

The detailed documentation for security headers can be found in [unrolled/secure](https://github.com/unrolled/secure#available-options).

## Configuration Examples

### Using Security Headers

Security-related headers (HSTS headers, Browser XSS filter, etc) can be managed using the SecureHeaders middleware.
This functionality makes it possible to easily use security features by adding headers on the request.

```yaml tab="Docker"
labels:
  - "traefik.http.middlewares.testSecureHeader.secureHeaders.framedeny=true"
  - "traefik.http.middlewares.testSecureHeader.secureHeaders.browserxssfilter=true"
```

```yaml tab="Kubernetes"
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-secure-header
spec:
  secureHeaders:
    frameDeny: true
    browserXssFilter: true
```

```yaml tab="Consul Catalog"
- "traefik.http.middlewares.testSecureHeader.secureHeaders.framedeny=true"
- "traefik.http.middlewares.testSecureHeader.secureHeaders.browserxssfilter=true"
```

```yaml tab="File (YAML)"
http:
  middlewares:
    testSecureHeader:
      secureHeaders:
        frameDeny: true
        browserXssFilter: true
```

```toml tab="File (TOML)"
[http.middlewares]
  [http.middlewares.testSecureHeader.secureHeaders]
    frameDeny = true
    browserXssFilter = true
```

### CORS Headers

CORS (Cross-Origin Resource Sharing) headers can be added and configured in a manner similar to the custom headers above.
This functionality allows for more advanced security features to quickly be set.
If CORS headers are set, then the middleware does not pass preflight requests to any service,
instead the response will be generated and sent back to the client directly.

```yaml tab="Docker"
labels:
  - "traefik.http.middlewares.testSecureHeader.secureHeaders.accesscontrolallowmethods=GET,OPTIONS,PUT"
  - "traefik.http.middlewares.testSecureHeader.secureHeaders.accesscontrolalloworiginlist=https://foo.bar.org,https://example.org"
  - "traefik.http.middlewares.testSecureHeader.secureHeaders.accesscontrolmaxage=100"
  - "traefik.http.middlewares.testSecureHeader.secureHeaders.addvaryheader=true"
```

```yaml tab="Kubernetes"
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-header
spec:
  secureHeaders:
    accessControlAllowMethods:
      - "GET"
      - "OPTIONS"
      - "PUT"
    accessControlAllowOriginList:
      - "https://foo.bar.org"
      - "https://example.org"
    accessControlMaxAge: 100
    addVaryHeader: true
```

```yaml tab="Consul Catalog"
- "traefik.http.middlewares.testSecureHeader.secureHeaders.accesscontrolallowmethods=GET,OPTIONS,PUT"
- "traefik.http.middlewares.testSecureHeader.secureHeaders.accesscontrolalloworiginlist=https://foo.bar.org,https://example.org"
- "traefik.http.middlewares.testSecureHeader.secureHeaders.accesscontrolmaxage=100"
- "traefik.http.middlewares.testSecureHeader.secureHeaders.addvaryheader=true"
```

```yaml tab="File (YAML)"
http:
  middlewares:
    testSecureHeader:
      secureHeaders:
        accessControlAllowMethods:
          - GET
          - OPTIONS
          - PUT
        accessControlAllowOriginList:
          - https://foo.bar.org
          - https://example.org
        accessControlMaxAge: 100
        addVaryHeader: true
```

```toml tab="File (TOML)"
[http.middlewares]
  [http.middlewares.testSecureHeader.secureHeaders]
    accessControlAllowMethods= ["GET", "OPTIONS", "PUT"]
    accessControlAllowOriginList = ["https://foo.bar.org","https://example.org"]
    accessControlMaxAge = 100
    addVaryHeader = true
```

## Configuration Options

### General

!!! warning

    Custom headers will overwrite existing headers if they have identical names.

### `accessControlAllowCredentials`

The `accessControlAllowCredentials` indicates whether the request can include user credentials.

### `accessControlAllowHeaders`

The `accessControlAllowHeaders` indicates which header field names can be used as part of the request.

### `accessControlAllowMethods`

The  `accessControlAllowMethods` indicates which methods can be used during requests.

### `accessControlAllowOriginList`

The `accessControlAllowOriginList` indicates whether a resource can be shared by returning different values.

A wildcard origin `*` can also be configured, and matches all requests.
If this value is set by a backend service, it will be overwritten by Traefik.

This value can contain a list of allowed origins.

More information including how to use the settings can be found at:

- [Mozilla.org](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin)
- [w3](https://fetch.spec.whatwg.org/#http-access-control-allow-origin)
- [IETF](https://tools.ietf.org/html/rfc6454#section-7.1)

Traefik no longer supports the `null` value, as it is [no longer recommended as a return value](https://w3c.github.io/webappsec-cors-for-developers/#avoid-returning-access-control-allow-origin-null).

### `accessControlAllowOriginListRegex`

The `accessControlAllowOriginListRegex` option is the counterpart of the `accessControlAllowOriginList` option with regular expressions instead of origin values.
It allows all origins that contain any match of a regular expression in the `accessControlAllowOriginList`.

!!! tip

    Regular expressions and replacements can be tested using online tools such as [Go Playground](https://play.golang.org/p/mWU9p-wk2ru) or the [Regex101](https://regex101.com/r/58sIgx/2).

    When defining a regular expression within YAML, any escaped character needs to be escaped twice: `example\.com` needs to be written as `example\\.com`.

### `accessControlExposeHeaders`

The `accessControlExposeHeaders` indicates which headers are safe to expose to the api of a CORS API specification.

### `accessControlMaxAge`

The `accessControlMaxAge` indicates how many seconds a preflight request can be cached for.

### `addVaryHeader`

The `addVaryHeader` is used in conjunction with `accessControlAllowOriginList` to determine whether the `Vary` header should be added or modified to demonstrate that server responses can differ based on the value of the origin header.

### `allowedHosts`

The `allowedHosts` option lists fully qualified domain names that are allowed.

### `hostsProxyHeaders`

The `hostsProxyHeaders` option is a set of header keys that may hold a proxied hostname value for the request.

### `sslProxyHeaders`

The `sslProxyHeaders` option is set of header keys with associated values that would indicate a valid HTTPS request.
It can be useful when using other proxies (example: `"X-Forwarded-Proto": "https"`).

### `stsSeconds`

The `stsSeconds` is the max-age of the `Strict-Transport-Security` header.
If set to `0`, the header is not set.

### `stsIncludeSubdomains`

If the `stsIncludeSubdomains` is set to `true`, the `includeSubDomains` directive is appended to the `Strict-Transport-Security` header.

### `stsPreload`

Set `stsPreload` to `true` to have the `preload` flag appended to the `Strict-Transport-Security` header.

### `forceSTSHeader`

Set `forceSTSHeader` to `true` to add the STS header even when the connection is HTTP.

### `frameDeny`

Set `frameDeny` to `true` to add the `X-Frame-Options` header with the value of `DENY`.

### `customFrameOptionsValue`

The `customFrameOptionsValue` allows the `X-Frame-Options` header value to be set with a custom value.
This overrides the `FrameDeny` option.

### `contentTypeNosniff`

Set `contentTypeNosniff` to true to add the `X-Content-Type-Options` header with the value `nosniff`.

### `browserXssFilter`

Set `browserXssFilter` to true to add the `X-XSS-Protection` header with the value `1; mode=block`.

### `customBrowserXSSValue`

The `customBrowserXssValue` option allows the `X-XSS-Protection` header value to be set with a custom value.
This overrides the `BrowserXssFilter` option.

### `contentSecurityPolicy`

The `contentSecurityPolicy` option allows the `Content-Security-Policy` header value to be set with a custom value.

### `publicKey`

The `publicKey` implements HPKP to prevent MITM attacks with forged certificates.

### `referrerPolicy`

The `referrerPolicy` allows sites to control whether browsers forward the `Referer` header to other sites.

### `permissionsPolicy`

The `permissionsPolicy` allows sites to control browser features.

### `isDevelopment`

Set `isDevelopment` to `true` when developing to mitigate the unwanted effects of the `AllowedHosts`, SSL, and STS options.
Usually testing takes place using HTTP, not HTTPS, and on `localhost`, not your production domain.
If you would like your development environment to mimic production with complete Host blocking, SSL redirects, and STS headers, leave this as `false`.

{!traefik-for-business-applications.md!}
