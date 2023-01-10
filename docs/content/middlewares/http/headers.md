---
title: "Traefik Headers Documentation"
description: "In Traefik Proxy, the HTTP headers middleware manages the headers of requests and responses. Read the technical documentation."
---

# Headers

Managing Request/Response Headers
{: .subtitle }

![Headers](../../assets/img/middleware/headers.png)

The Headers middleware manages the headers of requests and responses.

A set of forwarded headers are automatically added by default. See the [FAQ](../../getting-started/faq.md#what-are-the-forwarded-headers-when-proxying-http-requests) for more information.

## Configuration Examples

### Adding Headers to the Request and the Response

The following example adds the `X-Script-Name` header to the proxied request and the `X-Custom-Response-Header` header to the response

```yaml tab="Docker"
labels:
  - "traefik.http.middlewares.testHeader.headers.customrequestheaders.X-Script-Name=test"
  - "traefik.http.middlewares.testHeader.headers.customresponseheaders.X-Custom-Response-Header=value"
```

```yaml tab="Kubernetes"
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-header
spec:
  headers:
    customRequestHeaders:
      X-Script-Name: "test"
    customResponseHeaders:
      X-Custom-Response-Header: "value"
```

```yaml tab="Consul Catalog"
- "traefik.http.middlewares.testheader.headers.customrequestheaders.X-Script-Name=test"
- "traefik.http.middlewares.testheader.headers.customresponseheaders.X-Custom-Response-Header=value"
```

```yaml tab="File (YAML)"
http:
  middlewares:
    testHeader:
      headers:
        customRequestHeaders:
          X-Script-Name: "test"
        customResponseHeaders:
          X-Custom-Response-Header: "value"
```

```toml tab="File (TOML)"
[http.middlewares]
  [http.middlewares.testHeader.headers]
    [http.middlewares.testHeader.headers.customRequestHeaders]
        X-Script-Name = "test"
    [http.middlewares.testHeader.headers.customResponseHeaders]
        X-Custom-Response-Header = "value"
```

### Adding and Removing Headers

In the following example, requests are proxied with an extra `X-Script-Name` header while their `X-Custom-Request-Header` header gets stripped,
and responses are stripped of their `X-Custom-Response-Header` header.

```yaml tab="Docker"
labels:
  - "traefik.http.middlewares.testheader.headers.customrequestheaders.X-Script-Name=test"
  - "traefik.http.middlewares.testheader.headers.customrequestheaders.X-Custom-Request-Header="
  - "traefik.http.middlewares.testheader.headers.customresponseheaders.X-Custom-Response-Header="
```

```yaml tab="Kubernetes"
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: test-header
spec:
  headers:
    customRequestHeaders:
      X-Script-Name: "test" # Adds
      X-Custom-Request-Header: "" # Removes
    customResponseHeaders:
      X-Custom-Response-Header: "" # Removes
```

```yaml tab="Consul Catalog"
- "traefik.http.middlewares.testheader.headers.customrequestheaders.X-Script-Name=test"
- "traefik.http.middlewares.testheader.headers.customrequestheaders.X-Custom-Request-Header="
- "traefik.http.middlewares.testheader.headers.customresponseheaders.X-Custom-Response-Header="
```

```yaml tab="File (YAML)"
http:
  middlewares:
    testHeader:
      headers:
        customRequestHeaders:
          X-Script-Name: "test" # Adds
          X-Custom-Request-Header: "" # Removes
        customResponseHeaders:
          X-Custom-Response-Header: "" # Removes
```

```toml tab="File (TOML)"
[http.middlewares]
  [http.middlewares.testHeader.headers]
    [http.middlewares.testHeader.headers.customRequestHeaders]
        X-Script-Name = "test" # Adds
        X-Custom-Request-Header = "" # Removes
    [http.middlewares.testHeader.headers.customResponseHeaders]
        X-Custom-Response-Header = "" # Removes
```

## Configuration Options

### General

!!! warning

    Custom headers will overwrite existing headers if they have identical names.

### `customRequestHeaders`

The `customRequestHeaders` option lists the header names and values to apply to the request.

### `customResponseHeaders`

The `customResponseHeaders` option lists the header names and values to apply to the response.

{!traefik-for-business-applications.md!}
