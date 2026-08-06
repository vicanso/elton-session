package main

import (
	"bytes"
	"net/http"
	"strconv"
	"time"

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
	signedKeys.SetKeys([]string{
		"cuttlefish",
	})
	e.SignedKeys = signedKeys

	e.Use(session.NewByCookie(session.CookieConfig{
		Store:  store,
		Signed: true,
		TTL:    10 * time.Hour,
		GenID: func() string {
			// Prefer a strong unique id generator (e.g. xid/uuid) in production.
			return strconv.FormatInt(time.Now().UnixNano(), 10)
		},
		Cookie: http.Cookie{
			Name:   "jt",
			Path:   "/",
			MaxAge: 24 * 3600,
		},
	}))

	e.GET("/", func(c *elton.Context) error {
		se := session.MustFromContext(c)
		views := se.GetInt("views")
		_ = se.Set(c.Context(), "views", views+1)
		c.BodyBuffer = bytes.NewBufferString("hello world " + strconv.Itoa(views))
		return nil
	})

	if err = e.ListenAndServe(":3000"); err != nil {
		panic(err)
	}
}
