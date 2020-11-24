# elton-session

[![Build Status](https://img.shields.io/travis/vicanso/elton-session.svg?label=linux+build)](https://travis-ci.org/vicanso/elton-session)

Session middleware for elton, it support memory store by default.

Session id store by cookie is more simple. It also support by http header or other ways for session id. 

## NewByCookie

Get session id from cookie(signed). The first time commit session, it will add cookie to http response.

```go
package main

import (
	"bytes"
	"strconv"
	"time"

	"github.com/vicanso/elton"
	session "github.com/vicanso/elton-session"
)

func main() {
	store, err := session.NewMemoryStore(10)
	if err != nil {
		panic(err)
	}
	d := elton.New()
	signedKeys := &elton.RWMutexSignedKeys{}
	signedKeys.SetKeys([]string{
		"cuttlefish",
	})
	d.SignedKeys = signedKeys

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

	d.GET("/", func(c *elton.Context) (err error) {
		value, _ := c.Get(session.Key)
		se := value.(*session.Session)
		views := se.GetInt("views")
		se.Set("views", views+1)
		c.BodyBuffer = bytes.NewBufferString("hello world " + strconv.Itoa(views))
		return
	})

	d.ListenAndServe(":3000")
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

	"github.com/vicanso/elton"
	session "github.com/vicanso/elton-session"
)

func main() {
	store, err := session.NewMemoryStore(10)
	if err != nil {
		panic(err)
	}
	e := elton.New()
	signedKeys := &elton.RWMutexSignedKeys{}
	signedKeys.SetKeys([]string{
		"cuttlefish",
	})
	e.SignedKeys = signedKeys

	e.Use(session.NewByCookie(session.CookieConfig{
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

	e.GET("/", func(c *elton.Context) (err error) {
		se := c.Get(session.Key).(*session.Session)
		views := se.GetInt("views")
		_ = se.Set("views", views+1)
		c.BodyBuffer = bytes.NewBufferString("hello world " + strconv.Itoa(views))
		return
	})

	err = e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
```


## NewMemoryStore

Create a memory store for session.

- `size` max size of store

```go
store, err := NewMemoryStore(1024)
```

## NewMemoryStoreByConfig

Create a memory store for session.

- `config.Size` max size of store
- `config.SaveAs` save store sa file
- `config.Interval` flush to file's interval


```go
store, err := NewMemoryStore(MemoryStoreConfig{
	Size: 1024,
	SaveAs: "/tmp/elton-session-store",
	Interval: 60 * time.Second,
})
```

# Other store

You can use other store for session, like redis and mongodb.

```go
type (
	// RedisStore redis store for session
	RedisStore struct {
		client *redis.Client
		prefix string
	}
)

func (rs *RedisStore) getKey(key string) string {
	return rs.prefix + key
}

// Get get the session from redis
func (rs *RedisStore) Get(key string) ([]byte, error) {
	buf, err := rs.client.Get(rs.getKey(key)).Bytes()
	if err == redis.Nil {
		return buf, nil
	}
	return buf, err
}

// Set set the session to redis
func (rs *RedisStore) Set(key string, data []byte, ttl time.Duration) error {
	return rs.client.Set(rs.getKey(key), data, ttl).Err()
}

// Destroy remove the session from redis
func (rs *RedisStore) Destroy(key string) error {
	return rs.client.Del(rs.getKey(key)).Err()
}

// NewRedisStore create new redis store instance
func NewRedisStore(client *redis.Client, prefix string) *RedisStore {
	rs := &RedisStore{}
	rs.client = client
	rs.prefix = prefix
	return rs
}
```
