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
		se := c.Get(session.Key).(*session.Session)
		views := se.GetInt("views")
		_ = se.Set("views", views+1)
		c.BodyBuffer = bytes.NewBufferString("hello world " + strconv.Itoa(views))
		return
	})

	err = d.ListenAndServe(":3000")
	if err != nil {
		panic(err)
	}
}
