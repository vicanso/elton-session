# elton-session

[![license](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/vicanso/elton-session/blob/master/LICENSE)
[![Build Status](https://github.com/vicanso/elton-session/workflows/Test/badge.svg)](https://github.com/vicanso/elton-session/actions)

Session middleware for [elton](https://github.com/vicanso/elton) v2, with a built-in memory store.

Session IDs can be carried by cookie (optionally signed), HTTP header, or any custom Get/Set pair.

## Install

```bash
go get github.com/vicanso/elton-session/v2
```

Requires Go 1.24+.

## NewByCookie

Reads the session ID from a cookie. On first commit, writes `Set-Cookie` (HttpOnly is always forced).

```go
package main

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/vicanso/elton/v2"
	session "github.com/vicanso/elton-session/v2"
)

func main() {
	store, err := session.NewMemoryStore(10)
	if err != nil {
		panic(err)
	}
	e := elton.New()
	signedKeys := &elton.RWMutexSignedKeys{}
	signedKeys.SetKeys([]string{"cuttlefish"})
	e.SignedKeys = signedKeys

	e.Use(session.NewByCookie(session.CookieConfig{
		Store:  store,
		Signed: true,
		TTL:    10 * time.Hour,
		GenID: func() string {
			return strings.ToUpper(xid.New().String())
		},
		Cookie: http.Cookie{
			Name:   "jt",
			Path:   "/",
			MaxAge: 24 * 3600,
			Secure: true,
		},
	}))

	e.GET("/", func(c *elton.Context) error {
		se := session.MustFromContext(c)
		views := se.GetInt("views")
		_ = se.Set(c.Context(), "views", views+1)
		c.BodyBuffer = bytes.NewBufferString("hello world " + strconv.Itoa(views))
		return nil
	})

	if err := e.ListenAndServe(":3000"); err != nil {
		panic(err)
	}
}
```

## NewByHeader

Reads the session ID from a request header. On first commit, writes the same header on the response.

```go
e.Use(session.NewByHeader(session.HeaderConfig{
	Store: store,
	TTL:   10 * time.Hour,
	GenID: func() string {
		return strings.ToUpper(xid.New().String())
	},
	Name: "X-Session-ID",
}))
```

## New (custom ID transport)

```go
e.Use(session.New(session.Config{
	Store: store,
	TTL:   time.Hour,
	GenID: generateID,
	Get: func(c *elton.Context) (string, error) {
		// read id from query, cookie, header, ...
		return c.QueryParam("sid"), nil
	},
	Set: func(c *elton.Context, id string) error {
		// write id back however you need
		c.SetHeader("X-Session-ID", id)
		return nil
	},
}))
```

## Memory store

```go
store, err := session.NewMemoryStore(1024)

store, err := session.NewMemoryStoreByConfig(session.MemoryStoreConfig{
	Size:     1024,
	SaveAs:   "/tmp/elton-session-store",
	Interval: 60 * time.Second,
})
// When finished (e.g. process shutdown), stop background flush:
// store.StopFlush()
```

## Custom store

Implement `session.Store`:

```go
type Store interface {
	Get(ctx context.Context, id string) ([]byte, error)
	Set(ctx context.Context, id string, data []byte, ttl time.Duration) error
	Destroy(ctx context.Context, id string) error
}
```

Example Redis store:

```go
type RedisStore struct {
	client *redis.Client
	prefix string
}

func (rs *RedisStore) key(id string) string {
	return rs.prefix + id
}

func (rs *RedisStore) Get(ctx context.Context, id string) ([]byte, error) {
	buf, err := rs.client.Get(ctx, rs.key(id)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return buf, err
}

func (rs *RedisStore) Set(ctx context.Context, id string, data []byte, ttl time.Duration) error {
	return rs.client.Set(ctx, rs.key(id), data, ttl).Err()
}

func (rs *RedisStore) Destroy(ctx context.Context, id string) error {
	return rs.client.Del(ctx, rs.key(id)).Err()
}
```

## Session helpers

```go
se, ok := session.FromContext(c)
se = session.MustFromContext(c)

// typed getters (JSON-friendly casting)
se.GetString("account")
se.GetInt("views")
se.GetBool("admin")

// metadata
se.GetCreatedAt()
se.GetUpdatedAt()
se.GetExpiredAt()
```

## Breaking changes (v2)

- Module path: `github.com/vicanso/elton-session/v2`
- Depends on `github.com/vicanso/elton/v2` and `github.com/vicanso/hes` v1
- Config field `Expired` → `TTL`
- Context key constant `Key` → `DefaultKey`
- `Get` / `MustGet` → `FromContext` / `MustFromContext` (plus `FromContextKey`)
- `Session.GetData()` → `Session.Data()`
- `CookieConfig` cookie fields collapsed into `Cookie http.Cookie`
- Cookie mode always forces `HttpOnly=true`; empty `Path` defaults to `"/"`
- `LazyFetch` is honored by `NewByCookie` / `NewByHeader` (was ignored in v1)
- Optional `Config.Key` / `CookieConfig.Key` / `HeaderConfig.Key` for a custom context key
- Errors use hes v1 options API (`WithCategory` / `WithStatus` / `WithException`)
- `interface{}` → `any`
- Memory store flush: ticker is stopped on `StopFlush`; persistence uses `os.ReadFile` / `os.WriteFile`
- Go 1.24+
