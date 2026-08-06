// MIT License

// Copyright (c) 2020 Tree Xie

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton/v2"
	"github.com/vicanso/hes"
)

func generateID() string {
	b := make([]byte, 24)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func TestWrapError(t *testing.T) {
	assert := assert.New(t)
	err := wrapError(errors.New("abcd"))
	he, ok := hes.As(err)
	assert.True(ok)
	assert.True(he.Exception)
	assert.Equal(ErrCategory, he.Category)
}

func TestFetch(t *testing.T) {
	assert := assert.New(t)
	store, err := NewMemoryStore(10)
	ctx := context.Background()
	assert.Nil(err)
	s := Session{Store: store}
	_, err = s.Fetch(ctx)
	assert.Nil(err, "fetch fail")

	s = Session{
		Store: store,
		ID:    generateID(),
	}
	err = store.Set(ctx, s.ID, []byte(`{"a": 1}`), 10*time.Second)
	assert.Nil(err, "set store fail")
	_, err = s.Fetch(ctx)
	assert.Nil(err, "fetch fail")
	assert.Equal(1, s.GetInt("a"), "fetch data fail")

	err = store.Set(ctx, s.ID, []byte(`{"a": 1`), 10*time.Second)
	assert.Nil(err, "set store fail")
	s.fetched = false
	_, err = s.Fetch(ctx)
	assert.NotNil(err, "fetch not json data should return error")
}

func TestGetSetData(t *testing.T) {
	assert := assert.New(t)
	store, err := NewMemoryStore(10)
	ctx := context.Background()
	assert.Nil(err, "new memory store fail")
	s := Session{
		Store: store,
		ID:    generateID(),
	}
	err = s.Set(ctx, "", nil)
	assert.Nil(err, "set empty key shouldn't be fail")
	err = s.SetMap(ctx, nil)
	assert.Nil(err, "set empty map shouldn't be fail")
	_, err = s.Fetch(ctx)
	assert.Nil(err, "fetch fail")
	_ = s.SetMap(ctx, map[string]any{
		"a": 1,
		"b": "2",
	})
	err = s.Set(ctx, "a", nil)
	assert.Nil(err, "remove key fail")
	assert.Empty(s.Get("a"), "remove key fail")

	err = s.SetMap(ctx, map[string]any{
		"b": nil,
	})
	assert.Nil(err, "remove key fail")
	assert.Empty(s.Get("b"), "remove key fail")

	assert.False(s.Readonly())
	s.EnableReadonly()
	assert.True(s.Readonly())
	err = s.Set(ctx, "a", 1)
	assert.Equal(ErrIsReadonly, err)
	err = s.SetMap(ctx, map[string]any{
		"a": 1,
	})
	assert.Equal(ErrIsReadonly, err)
}

func TestCommit(t *testing.T) {
	assert := assert.New(t)
	store, err := NewMemoryStore(10)
	ctx := context.Background()
	assert.Nil(err, "new memory store fail")
	ttl := time.Second
	s := Session{Store: store}
	err = s.Commit(ctx, ttl)
	assert.Nil(err, "commit not modified fail")

	_, err = s.Fetch(ctx)
	assert.Nil(err)
	_ = s.Set(ctx, "a", 1)
	err = s.Commit(ctx, ttl)
	assert.Equal(ErrIDNil, err, "nil id commit should return error")

	s.ID = generateID()
	err = s.Commit(ctx, ttl)
	assert.Nil(err, "session commit fail")
	err = s.Commit(ctx, ttl)
	assert.Equal(ErrDuplicateCommit, err, "duplicate commit should return error")
}

func TestIgnoreModified(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	store, err := NewMemoryStore(10)
	assert.Nil(err, "new memory store fail")
	ttl := time.Second
	id := "id"
	s := Session{Store: store, ID: id}
	err = s.Set(ctx, "a", "abc")
	assert.Nil(err)
	err = s.Commit(ctx, ttl)
	assert.Nil(err)
	data, err := store.Get(ctx, id)
	assert.Nil(err)
	assert.True(strings.Contains(string(data), `"a":"abc"`))

	s1 := Session{Store: store, ID: id}
	err = s1.Set(ctx, "a", "def")
	assert.Nil(err)
	assert.True(s1.Modified())
}

func TestSession(t *testing.T) {
	assert := assert.New(t)
	store, err := NewMemoryStore(10)
	ctx := context.Background()
	assert.Nil(err, "new memory store fail")
	s := Session{
		ID:    generateID(),
		Store: store,
	}
	_, err = s.Fetch(ctx)
	assert.Nil(err, "fetch session fail")

	_, err = s.Fetch(ctx)
	assert.Nil(err, "fetch session twice fail")
	_ = s.Set(ctx, "a", "1")
	_ = s.SetMap(ctx, map[string]any{
		"b": 2,
		"c": true,
		"d": 1.1,
		"e": []string{"1", "2"},
	})

	assert.True(s.GetBool("c"))
	assert.Equal("1", s.Get("a").(string))
	assert.Equal("1", s.GetString("a"))
	assert.Equal(2, s.GetInt("b"))
	assert.Equal(1.1, s.GetFloat64("d"))
	assert.Equal([]string{"1", "2"}, s.GetStringSlice("e"))

	assert.NotEmpty(s.GetCreatedAt(), "get create time fail")
	assert.NotEmpty(s.GetUpdatedAt(), "get update time fail")

	assert.Equal("1", s.Data()["a"].(string), "get data fail")

	err = s.Commit(ctx, 10*time.Second)
	assert.Nil(err, "commit session fail")
	buf, _ := store.Get(ctx, s.ID)
	assert.NotEmpty(buf, "store shouldn't be empty after commit")

	updatedAt := s.GetUpdatedAt()
	time.Sleep(1 * time.Second)
	err = s.Refresh(ctx)
	assert.Nil(err)
	assert.NotEqual(updatedAt, s.GetUpdatedAt(), "refresh fail")

	err = s.Destroy(ctx)
	assert.Nil(err, "destroy session fail")

	buf, err = store.Get(ctx, s.ID)
	assert.Nil(err)
	assert.Empty(buf, "store should be empty after destroy")

	s.ID = ""
	err = s.Destroy(ctx)
	assert.Nil(err, "no session is should destroy success")
}

func TestNotFetchError(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	s := Session{}
	err := s.Set(ctx, "a", 1)
	assert.Nil(err)
	assert.Equal(1, s.GetInt("a"))

	s = Session{}
	err = s.SetMap(ctx, map[string]any{
		"a": 2,
	})
	assert.Nil(err)
	assert.Equal(2, s.GetInt("a"))

	s = Session{}
	err = s.Refresh(ctx)
	assert.Nil(err)
	assert.NotEmpty(s.GetUpdatedAt())
}

func TestSessionFromContext(t *testing.T) {
	assert := assert.New(t)
	c := elton.NewContext(nil, nil)
	se, exists := FromContext(c)
	assert.False(exists)
	assert.Nil(se)

	sess := &Session{}
	c.Set(DefaultKey, sess)
	se, exists = FromContext(c)
	assert.True(exists)
	assert.Equal(sess, se)

	se = MustFromContext(c)
	assert.Equal(sess, se)

	customKey := "my-session"
	c2 := elton.NewContext(nil, nil)
	c2.Set(customKey, sess)
	se, exists = FromContextKey(c2, customKey)
	assert.True(exists)
	assert.Equal(sess, se)
}

func TestSessionMiddleware(t *testing.T) {
	uid := "abcd"
	idName := "jt"
	ctx := context.Background()
	store, err := NewMemoryStore(10)
	assert.Nil(t, err, "new memory store fail")

	cookieSessionMiddleware := NewByCookie(CookieConfig{
		Store: store,
		TTL:   10 * time.Millisecond,
		GenID: func() string {
			return uid
		},
		Cookie: http.Cookie{
			Name:   idName,
			Path:   "/",
			Domain: "abc.com",
			MaxAge: 60,
			Secure: true,
		},
	})

	t.Run("session by cookie", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "/users/me", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)

		c.Next = func() error {
			se := MustFromContext(c)
			return se.Set(ctx, "foo", "bar")
		}
		err = cookieSessionMiddleware(c)
		assert.Nil(err, "session by cookie middleware fail")
		assert.Equal([]string{
			"jt=abcd; Path=/; Domain=abc.com; Max-Age=60; HttpOnly; Secure",
		}, c.Header()["Set-Cookie"], "set cookie fail")

		req = httptest.NewRequest("GET", "/users/me", nil)
		req.AddCookie(&http.Cookie{
			Name:  idName,
			Value: uid,
		})
		resp = httptest.NewRecorder()
		c = elton.NewContext(resp, req)
		c.Next = func() error {
			se := MustFromContext(c)
			if se.GetString("foo") != "bar" {
				return errors.New("get session fail")
			}
			return nil
		}
		err = cookieSessionMiddleware(c)
		assert.Nil(err, "session by cookie middleware fail")
	})

	headerSessionMiddleware := NewByHeader(HeaderConfig{
		Store: store,
		TTL:   10 * time.Millisecond,
		GenID: func() string {
			return uid
		},
		Name: idName,
	})

	t.Run("session by header", func(t *testing.T) {
		assert := assert.New(t)

		req := httptest.NewRequest("GET", "/users/me", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.Next = func() error {
			se := MustFromContext(c)
			return se.Set(ctx, "foo", "bar")
		}
		err = headerSessionMiddleware(c)
		assert.Nil(err, "session by header middleware fail")
		assert.Equal(uid, c.GetHeader(idName), "set header value fail")

		req = httptest.NewRequest("GET", "/users/me", nil)
		req.Header.Set(idName, uid)

		resp = httptest.NewRecorder()
		c = elton.NewContext(resp, req)
		c.Next = func() error {
			se := MustFromContext(c)
			if se.GetString("foo") != "bar" {
				return errors.New("get session fail")
			}
			return nil
		}
		err = headerSessionMiddleware(c)
		assert.Nil(err, "session by header middleware fail")
	})
}

func TestLazyFetch(t *testing.T) {
	assert := assert.New(t)
	store, err := NewMemoryStore(10)
	assert.Nil(err)
	ctx := context.Background()
	id := generateID()
	assert.Nil(store.Set(ctx, id, []byte(`{"views":3}`), time.Minute))

	mw := NewByHeader(HeaderConfig{
		Store:     store,
		TTL:       time.Minute,
		LazyFetch: true,
		GenID:     generateID,
		Name:      "X-Session-ID",
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Session-ID", id)
	resp := httptest.NewRecorder()
	c := elton.NewContext(resp, req)
	c.Next = func() error {
		se := MustFromContext(c)
		assert.False(se.fetched)
		assert.Equal(3, se.GetInt("views"))
		assert.True(se.fetched)
		return nil
	}
	assert.Nil(mw(c))
}


