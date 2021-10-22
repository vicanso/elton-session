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
			// 使用时需要使用uuid等生成唯一id
			return strconv.Itoa(int(time.Now().UnixNano()))
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
