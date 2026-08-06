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

// Package session is session middleware for elton v2.
//
// It loads session data from a pluggable Store, exposes a typed Session API on
// the request context, and commits modified data after the handler chain when
// the session was written.
//
// Session IDs can be carried by signed/unsigned cookies (NewByCookie), request
// headers (NewByHeader), or any custom Get/Set pair (New).
package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/spf13/cast"
	"github.com/vicanso/elton/v2"
	"github.com/vicanso/hes"
)

const (
	// CreatedAt is the metadata key for session creation time (RFC3339).
	CreatedAt = "_createdAt"
	// UpdatedAt is the metadata key for last modification time (RFC3339).
	UpdatedAt = "_updatedAt"
	// ExpiredAt is the metadata key for store expiry time (RFC3339), set on Commit.
	ExpiredAt = "_expiredAt"
	// ErrCategory is the hes error category for this middleware.
	ErrCategory = "elton-session"
	// DefaultKey is the default context key for *Session.
	DefaultKey = "_session"
)

var (
	// ErrDuplicateCommit is returned when Commit is called more than once
	// after the session was modified.
	ErrDuplicateCommit = newError("duplicate commit")
	// ErrIDNil is returned when Commit is called without a session ID.
	ErrIDNil = newError("session id is nil")
	// ErrIsReadonly is returned when Set/SetMap is called on a readonly session.
	ErrIsReadonly = newError("session is readonly")
)

// M is the session data map type.
type M map[string]any

// Store persists session payloads keyed by session ID.
type Store interface {
	// Get returns session bytes for id. Missing keys should return (nil, nil).
	Get(ctx context.Context, id string) ([]byte, error)
	// Set stores data with the given TTL.
	Set(ctx context.Context, id string, data []byte, ttl time.Duration) error
	// Destroy removes the session for id.
	Destroy(ctx context.Context, id string) error
}

// Config is the generic session middleware configuration.
// Get/Set own how the session ID is read from and written to the request/response.
type Config struct {
	// Store is required.
	Store Store
	// Skipper skips this middleware when it returns true.
	Skipper elton.Skipper
	// TTL is the store max age and is required (must be > 0).
	TTL time.Duration
	// LazyFetch defers loading session data until the first read/write.
	LazyFetch bool
	// GenID generates a new session ID when the session is first committed.
	// Required.
	GenID func() string
	// Get returns the session ID for the request. Empty string means no ID yet.
	// Required.
	Get func(c *elton.Context) (string, error)
	// Set writes a newly generated session ID to the response. Required.
	Set func(c *elton.Context, id string) error
	// Key is the context key for *Session. Default: DefaultKey.
	Key string
}

// CookieConfig configures session ID transport via HTTP cookies.
type CookieConfig struct {
	// Store is required.
	Store Store
	// Skipper skips this middleware when it returns true.
	Skipper elton.Skipper
	// TTL is the store max age and is required (must be > 0).
	TTL time.Duration
	// LazyFetch defers loading session data until the first read/write.
	LazyFetch bool
	// GenID generates a new session ID when the session is first committed.
	// Required.
	GenID func() string
	// Signed enables signed cookies (requires elton.SignedKeys on the engine).
	Signed bool
	// Cookie is the cookie template. Cookie.Name is required.
	// HttpOnly is forced to true; empty Path defaults to "/".
	Cookie http.Cookie
	// Key is the context key for *Session. Default: DefaultKey.
	Key string
}

// HeaderConfig configures session ID transport via a request/response header.
type HeaderConfig struct {
	// Store is required.
	Store Store
	// Skipper skips this middleware when it returns true.
	Skipper elton.Skipper
	// TTL is the store max age and is required (must be > 0).
	TTL time.Duration
	// LazyFetch defers loading session data until the first read/write.
	LazyFetch bool
	// GenID generates a new session ID when the session is first committed.
	// Required.
	GenID func() string
	// Name is the request header to read and the response header to write.
	// Required.
	Name string
	// Key is the context key for *Session. Default: DefaultKey.
	Key string
}

// Session holds per-request session state and provides typed accessors.
type Session struct {
	// Store is the session store.
	Store Store
	// ID is the session identifier. Empty until first commit (or when provided by client).
	ID string

	data      M
	fetched   bool
	modified  bool
	committed bool
	readonly  bool
}

func newError(message string) *hes.Error {
	return hes.New(
		message,
		hes.WithCategory(ErrCategory),
		hes.WithStatus(http.StatusInternalServerError),
		hes.WithException(),
	)
}

func wrapError(err error) error {
	if err == nil {
		return nil
	}
	return hes.Wrap(
		err,
		hes.WithCategory(ErrCategory),
		hes.WithStatus(http.StatusInternalServerError),
		hes.WithException(),
	)
}

func initMap() M {
	return M{
		CreatedAt: time.Now().Format(time.RFC3339),
	}
}

func (s *Session) fetch(ctx context.Context) error {
	if s.fetched {
		return nil
	}
	var buf []byte
	if s.ID != "" {
		b, err := s.Store.Get(ctx, s.ID)
		if err != nil {
			return err
		}
		buf = b
	}
	m := make(M)
	if len(buf) == 0 {
		m = initMap()
	} else {
		if err := json.Unmarshal(buf, &m); err != nil {
			return err
		}
	}
	s.fetched = true
	s.data = m
	return nil
}

// Fetch loads session data from the store (no-op if already fetched).
func (s *Session) Fetch(ctx context.Context) (M, error) {
	if s.fetched {
		return s.data, nil
	}
	if err := s.fetch(ctx); err != nil {
		return nil, err
	}
	return s.data, nil
}

// Destroy removes the session from the store and resets in-memory state.
func (s *Session) Destroy(ctx context.Context) error {
	if s.ID == "" {
		s.data = initMap()
		s.fetched = true
		s.modified = false
		s.committed = false
		return nil
	}
	if err := s.Store.Destroy(ctx, s.ID); err != nil {
		return err
	}
	s.ID = ""
	s.data = initMap()
	s.fetched = true
	s.modified = false
	s.committed = false
	return nil
}

func (s *Session) touchUpdatedAt() {
	s.data[UpdatedAt] = time.Now().Format(time.RFC3339)
	s.modified = true
}

// Set sets a single key. A nil value deletes the key.
func (s *Session) Set(ctx context.Context, key string, value any) error {
	if key == "" {
		return nil
	}
	return s.SetMap(ctx, map[string]any{key: value})
}

// SetMap merges values into the session. Nil values delete keys.
func (s *Session) SetMap(ctx context.Context, value map[string]any) error {
	if s.readonly {
		return ErrIsReadonly
	}
	if value == nil {
		return nil
	}
	if err := s.fetch(ctx); err != nil {
		return err
	}
	for k, v := range value {
		if v == nil {
			delete(s.data, k)
			continue
		}
		s.data[k] = v
	}
	s.touchUpdatedAt()
	return nil
}

// Readonly reports whether the session rejects writes.
func (s *Session) Readonly() bool {
	return s.readonly
}

// EnableReadonly marks the session as read-only for the rest of the request.
func (s *Session) EnableReadonly() {
	s.readonly = true
}

// Refresh updates UpdatedAt without changing other fields (marks modified).
func (s *Session) Refresh(ctx context.Context) error {
	if err := s.fetch(ctx); err != nil {
		return err
	}
	s.touchUpdatedAt()
	return nil
}

// Get returns a raw value by key. It lazy-fetches with context.Background()
// when data has not been loaded yet; prefer Fetch(ctx) in new code when you
// need cancellation.
func (s *Session) Get(key string) any {
	_ = s.fetch(context.Background())
	return s.data[key]
}

// GetBool returns a bool value by key.
func (s *Session) GetBool(key string) bool {
	return cast.ToBool(s.Get(key))
}

// GetString returns a string value by key.
func (s *Session) GetString(key string) string {
	return cast.ToString(s.Get(key))
}

// GetInt returns an int value by key.
func (s *Session) GetInt(key string) int {
	return cast.ToInt(s.Get(key))
}

// GetFloat64 returns a float64 value by key.
func (s *Session) GetFloat64(key string) float64 {
	return cast.ToFloat64(s.Get(key))
}

// GetStringSlice returns a []string value by key.
func (s *Session) GetStringSlice(key string) []string {
	return cast.ToStringSlice(s.Get(key))
}

// GetCreatedAt returns the session creation timestamp (RFC3339 string).
func (s *Session) GetCreatedAt() string {
	return cast.ToString(s.Get(CreatedAt))
}

// GetUpdatedAt returns the last modification timestamp (RFC3339 string).
func (s *Session) GetUpdatedAt() string {
	return cast.ToString(s.Get(UpdatedAt))
}

// GetExpiredAt returns the store expiry timestamp (RFC3339 string).
func (s *Session) GetExpiredAt() string {
	return cast.ToString(s.Get(ExpiredAt))
}

// Data returns the underlying session map (shared; do not mutate casually).
func (s *Session) Data() M {
	return s.data
}

// Modified reports whether the session has uncommitted writes.
func (s *Session) Modified() bool {
	return s.modified
}

// Commit writes modified session data to the store.
// No-op when unmodified. Returns ErrDuplicateCommit on a second commit after a write.
func (s *Session) Commit(ctx context.Context, ttl time.Duration) error {
	if !s.modified {
		return nil
	}
	if s.committed {
		return ErrDuplicateCommit
	}
	if s.ID == "" {
		return ErrIDNil
	}
	s.data[ExpiredAt] = time.Now().Add(ttl).Format(time.RFC3339)

	buf, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	if err = s.Store.Set(ctx, s.ID, buf, ttl); err != nil {
		return err
	}
	s.committed = true
	return nil
}

// FromContext returns the *Session stored on the context.
func FromContext(c *elton.Context) (*Session, bool) {
	value, ok := c.Get(DefaultKey)
	if !ok {
		return nil, false
	}
	s, ok := value.(*Session)
	if !ok {
		return nil, false
	}
	return s, true
}

// FromContextKey returns the *Session stored under a custom context key.
func FromContextKey(c *elton.Context, key string) (*Session, bool) {
	if key == "" {
		key = DefaultKey
	}
	value, ok := c.Get(key)
	if !ok {
		return nil, false
	}
	s, ok := value.(*Session)
	if !ok {
		return nil, false
	}
	return s, true
}

// MustFromContext returns the *Session or panics if missing.
func MustFromContext(c *elton.Context) *Session {
	se, ok := FromContext(c)
	if !ok || se == nil {
		panic("session is nil")
	}
	return se
}

func resolveKey(key string) string {
	if key == "" {
		return DefaultKey
	}
	return key
}

// New creates session middleware from Config.
func New(config Config) elton.Handler {
	store := config.Store
	getID := config.Get
	setID := config.Set
	genID := config.GenID
	ttl := config.TTL
	if store == nil || getID == nil || setID == nil || genID == nil || ttl <= 0 {
		panic(errors.New("session requires store, get, set, genID and positive TTL"))
	}
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	contextKey := resolveKey(config.Key)

	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		if _, exists := FromContextKey(c, contextKey); exists {
			return c.Next()
		}

		s := &Session{Store: store}
		id, err := getID(c)
		if err != nil {
			return wrapError(err)
		}
		if id != "" {
			s.ID = id
		}
		if !config.LazyFetch {
			if err = s.fetch(c.Context()); err != nil {
				return wrapError(err)
			}
		}

		c.Set(contextKey, s)
		if err = c.Next(); err != nil {
			return err
		}
		if !s.modified {
			return nil
		}
		if s.ID == "" {
			uid := genID()
			if err = setID(c, uid); err != nil {
				return wrapError(err)
			}
			s.ID = uid
		}
		if err = s.Commit(c.Context(), ttl); err != nil {
			return wrapError(err)
		}
		return nil
	}
}

// NewByCookie creates session middleware that stores the session ID in a cookie.
func NewByCookie(config CookieConfig) elton.Handler {
	if config.Cookie.Name == "" {
		panic(errors.New("session cookie name is required"))
	}
	template := config.Cookie
	template.HttpOnly = true
	if template.Path == "" {
		template.Path = "/"
	}
	signed := config.Signed
	name := template.Name

	getID := func(c *elton.Context) (string, error) {
		getCookie := c.Cookie
		if signed {
			getCookie = c.SignedCookie
		}
		// Missing cookie is not an error for session bootstrap.
		cookie, _ := getCookie(name)
		if cookie == nil {
			return "", nil
		}
		return cookie.Value, nil
	}
	setID := func(c *elton.Context, id string) error {
		cookie := template
		cookie.Value = id
		if signed {
			c.AddSignedCookie(&cookie)
		} else {
			c.AddCookie(&cookie)
		}
		return nil
	}

	return New(Config{
		Store:     config.Store,
		Skipper:   config.Skipper,
		TTL:       config.TTL,
		LazyFetch: config.LazyFetch,
		GenID:     config.GenID,
		Get:       getID,
		Set:       setID,
		Key:       config.Key,
	})
}

// NewByHeader creates session middleware that stores the session ID in a header.
func NewByHeader(config HeaderConfig) elton.Handler {
	if config.Name == "" {
		panic(errors.New("session header name is required"))
	}
	name := config.Name
	getID := func(c *elton.Context) (string, error) {
		return c.GetRequestHeader(name), nil
	}
	setID := func(c *elton.Context, id string) error {
		c.SetHeader(name, id)
		return nil
	}
	return New(Config{
		Store:     config.Store,
		Skipper:   config.Skipper,
		TTL:       config.TTL,
		LazyFetch: config.LazyFetch,
		GenID:     config.GenID,
		Get:       getID,
		Set:       setID,
		Key:       config.Key,
	})
}
