# cod-session

[![Build Status](https://img.shields.io/travis/vicanso/cod-session.svg?label=linux+build)](https://travis-ci.org/vicanso/cod-session)

session middleware for cod.

## NewByCookie

Get session id from cookie(signed). The first time commit session, it will add cookie to http response.

```go
package main

import (
	"bytes"
	"strconv"
	"time"

	"github.com/vicanso/cod"
	session "github.com/vicanso/cod-session"
)

func main() {
	store, err := session.NewMemoryStore(10)
	if err != nil {
		panic(err)
	}
	d := cod.New()
	d.Keys = []string{
		"cuttlefish",
	}

	d.Use(session.NewByCookie(session.CookieConfig{
		Store:   store,
		Signed:  true,
		Expired: 10 * time.Hour,
		GenID: func() string {
			// suggest to use uuid function
			return strconv.FormatInt(time.Now().UnixNano(), 34)
		},
		Name:     "jt",
		Path:     "/",
		MaxAge:   24 * 3600,
		HttpOnly: true,
	}))

	d.GET("/", func(c *cod.Context) (err error) {
		se := c.Get(session.Key).(*session.Session)
		views := se.GetInt("views")
		se.Set("views", views+1)
		c.BodyBuffer = bytes.NewBufferString("hello world " + strconv.Itoa(views))
		return
	})

	d.ListenAndServe(":7001")
}
```

## NewByHeader

Get session id from http request header. The first time commit session, it will add a response's header to http response.

```go
package main

import (
	"bytes"
	"strconv"
	"time"

	"github.com/vicanso/cod"
	session "github.com/vicanso/cod-session"
)

func main() {
	store, err := session.NewMemoryStore(10)
	if err != nil {
		panic(err)
	}
	d := cod.New()
	d.Keys = []string{
		"cuttlefish",
	}

	d.Use(session.NewByHeader(session.HeaderConfig{
		Store:   store,
		Expired: 10 * time.Hour,
		GenID: func() string {
			// suggest to use uuid function
			return strconv.FormatInt(time.Now().UnixNano(), 34)
		},
		// header's name
		Name: "jt",
	}))

	d.GET("/", func(c *cod.Context) (err error) {
		se := c.Get(session.Key).(*session.Session)
		views := se.GetInt("views")
		se.Set("views", views+1)
		c.BodyBuffer = bytes.NewBufferString("hello world " + strconv.Itoa(views))
		return
	})

	d.ListenAndServe(":7001")
}
```
