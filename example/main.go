package main

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/rs/xid"
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
			return strings.ToUpper(xid.New().String())
		},
		Name:     "jt",
		Path:     "/",
		MaxAge:   24 * 3600,
		HttpOnly: true,
	}))

	e.GET("/", func(c *elton.Context) (err error) {
		value, _ := c.Get(session.Key)
		se := value.(*session.Session)
		views := se.GetInt("views")
		_ = se.Set(c.Context(), "views", views+1)
		c.BodyBuffer = bytes.NewBufferString("hello world " + strconv.Itoa(views))
		return
	})

	err = e.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
