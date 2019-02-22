package session

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vicanso/cod"
	"github.com/vicanso/hes"
)

func TestWrapError(t *testing.T) {
	err := errors.New("abcd")
	err = wrapError(err)
	he, _ := err.(*hes.Error)
	if !he.Exception ||
		he.Category != ErrCategorySession {
		t.Fatalf("wrap error fail")
	}
}

func TestFetch(t *testing.T) {
	store, err := NewMemoryStore(10)
	if err != nil {
		t.Fatalf("new memory store fail, %v", err)
	}
	s := Session{
		Store: store,
	}
	_, err = s.Fetch()
	if err != nil {
		t.Fatalf("fetch fail, %v", err)
	}
	s = Session{
		Store: store,
		ID:    generateID(),
	}
	err = store.Set(s.ID, []byte(`{"a": 1}`), 10*time.Second)
	if err != nil {
		t.Fatalf("set store fail, %v", err)
	}
	_, err = s.Fetch()
	if err != nil {
		t.Fatalf("fetch fail, %v", err)
	}
	if s.GetInt("a") != 1 {
		t.Fatalf("fetch data fail")
	}
}

func TestGetSetData(t *testing.T) {
	store, err := NewMemoryStore(10)
	if err != nil {
		t.Fatalf("new memory store fail, %v", err)
	}
	s := Session{
		Store: store,
		ID:    generateID(),
	}
	err = s.Set("", nil)
	if err != nil {
		t.Fatalf("set empty key should not be fail")
	}
	err = s.SetMap(nil)
	if err != nil {
		t.Fatalf("set empty map should not be fail")
	}
	_, err = s.Fetch()
	s.SetMap(map[string]interface{}{
		"a": 1,
		"b": "2",
	})
	err = s.Set("a", nil)
	if err != nil || s.Get("a") != nil {
		t.Fatalf("remove key fail, %v", err)
	}
	err = s.SetMap(map[string]interface{}{
		"b": nil,
	})
	if err != nil || s.Get("b") != nil {
		t.Fatalf("remove key fail, %v", err)
	}
}

func TestCommit(t *testing.T) {
	store, err := NewMemoryStore(10)
	if err != nil {
		t.Fatalf("new memory store fail, %v", err)
	}
	ttl := time.Second
	s := Session{
		Store: store,
	}
	err = s.Commit(ttl)
	if err != nil {
		t.Fatalf("commit not modified fail, %v", err)
	}
	s.Fetch()
	s.Set("a", 1)
	err = s.Commit(ttl)
	if err != ErrIDNil {
		t.Fatalf("nil id commit should return error")
	}
	s.ID = generateID()
	err = s.Commit(ttl)
	if err != nil {
		t.Fatalf("session commit fail, %v", err)
	}
	err = s.Commit(ttl)
	if err != ErrDuplicateCommit {
		t.Fatalf("duplicate commit should return error")
	}
}

func TestSession(t *testing.T) {
	store, err := NewMemoryStore(10)
	if err != nil {
		t.Fatalf("new memory store fail, %v", err)
	}
	s := Session{
		ID:    generateID(),
		Store: store,
	}
	_, err = s.Fetch()
	if err != nil {
		t.Fatalf("fetch session fail, %v", err)
	}
	_, err = s.Fetch()
	if err != nil {
		t.Fatalf("fetch session twice fail, %v", err)
	}
	s.Set("a", "1")
	s.SetMap(map[string]interface{}{
		"b": 2,
		"c": true,
		"d": 1.1,
		"e": []string{"1", "2"},
	})

	if !s.GetBool("c") ||
		s.Get("a").(string) != "1" ||
		s.GetString("a") != "1" ||
		s.GetInt("b") != 2 ||
		s.GetFloat64("d") != 1.1 ||
		strings.Join(s.GetStringSlice("e"), ",") != "1,2" {
		t.Fatalf("get data from session fail")
	}

	if s.GetCreatedAt() == "" ||
		s.GetUpdatedAt() == "" {
		t.Fatalf("get create or update time fail")
	}

	if s.GetData()["a"].(string) != "1" {
		t.Fatalf("get data fail")
	}
	err = s.Commit(10 * time.Second)
	if err != nil {
		t.Fatalf("commit session fail, %v", err)
	}
	buf, _ := store.Get(s.ID)
	if len(buf) == 0 {
		t.Fatalf("store should not be empty after commit")
	}

	updatedAt := s.GetUpdatedAt()
	time.Sleep(1 * time.Second)
	s.Refresh()
	if s.GetUpdatedAt() == updatedAt {
		t.Fatalf("refresh fail")
	}

	err = s.Destroy()
	if err != nil {
		t.Fatalf("destroy session fail, %v", err)
	}
	buf, err = store.Get(s.ID)
	if err != nil ||
		len(buf) != 0 {
		t.Fatalf("store should not be empty after destroy")
	}
}

func TestNotFetchError(t *testing.T) {
	s := Session{}
	err := s.Set("a", 1)
	if err != ErrNotFetched {
		t.Fatalf("should return not fetch error")
	}

	err = s.SetMap(map[string]interface{}{
		"a": 1,
	})
	if err != ErrNotFetched {
		t.Fatalf("should return not fetch error")
	}

	err = s.Refresh()
	if err != ErrNotFetched {
		t.Fatalf("should return not fetch error")
	}

	if s.Get("a") != nil {
		t.Fatalf("should return nil before fetch")
	}
}

func TestSessionMiddleware(t *testing.T) {
	d := &cod.Cod{
		Keys: []string{
			"secret",
			"cuttlefisk",
		},
	}
	store, err := NewMemoryStore(10)
	if err != nil {
		t.Fatalf("new memory store fail, %v", err)
	}
	fn := NewSessionWithCookie(CookieConfig{
		Store:   store,
		Signed:  true,
		Expired: 10 * time.Millisecond,
		GenID: func() string {
			return "abcd"
		},
		Name:     "jt",
		Path:     "/",
		Domain:   "abc.com",
		MaxAge:   60,
		Secure:   true,
		HttpOnly: true,
	})
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	c := cod.NewContext(resp, req)
	// 必须 cod 实例有配置密钥才会生成 signed cookie
	c.Cod(d)
	c.Next = func() error {
		se := c.Get(SessionKey).(*Session)
		se.Set("foo", "bar")
		return nil
	}
	err = fn(c)
	if err != nil {
		t.Fatalf("session middleware fail, %v", err)
	}
	if strings.Join(c.Headers["Set-Cookie"], ",") != "jt=abcd; Path=/; Domain=abc.com; Max-Age=60; HttpOnly; Secure,jt.sig=sE80Oh3EoVzvllgRnFRBHy5As0U; Path=/; Domain=abc.com; Max-Age=60; HttpOnly; Secure" {
		t.Fatalf("set signed cookie fail")
	}

	req = httptest.NewRequest("GET", "/users/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  "jt",
		Value: "abcd",
	})
	req.AddCookie(&http.Cookie{
		Name:  "jt.sig",
		Value: "sE80Oh3EoVzvllgRnFRBHy5As0U",
	})
	resp = httptest.NewRecorder()
	c = cod.NewContext(resp, req)
	c.Cod(d)
	c.Next = func() error {
		se := c.Get(SessionKey).(*Session)
		if se.GetString("foo") != "bar" {
			return errors.New("get session fail")
		}
		return nil
	}
	err = fn(c)
	if err != nil {
		t.Fatalf("session middleware fail, %v", err)
	}
}
