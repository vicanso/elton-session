package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStore(t *testing.T) {
	key := generateID()
	data := []byte("tree.xie")
	ttl := 300 * time.Second
	ms, _ := NewMemoryStore(1024)

	t.Run("not init", func(t *testing.T) {
		assert := assert.New(t)
		tmp := &MemoryStore{}
		_, err := tmp.Get(key)
		assert.Equal(err, ErrNotInit)

		err = tmp.Set(key, data, ttl)
		assert.Equal(err, ErrNotInit)

		err = tmp.Destroy(key)
		assert.Equal(err, ErrNotInit)
	})

	t.Run("get not exists data", func(t *testing.T) {
		assert := assert.New(t)
		buf, err := ms.Get(key)

		assert.Nil(err)
		assert.Empty(buf, "should return empty bytes")
	})

	t.Run("set data", func(t *testing.T) {
		assert := assert.New(t)
		err := ms.Set(key, data, ttl)
		assert.Nil(err)
		buf, err := ms.Get(key)
		assert.Nil(err)
		assert.Equal(data, buf, "the data isn't the same after set")
	})

	t.Run("destroy", func(t *testing.T) {
		assert := assert.New(t)
		err := ms.Destroy(key)
		assert.Nil(err)
		buf, err := ms.Get(key)
		assert.Nil(err)
		assert.Empty(buf, "should return empty bytes after destroy")
	})

	t.Run("expired", func(t *testing.T) {
		assert := assert.New(t)
		err := ms.Set(key, data, 0)
		assert.Nil(err)
		time.Sleep(time.Second)
		buf, err := ms.Get(key)
		assert.Nil(err)
		assert.Empty(buf, "expired data should be nil")
	})

	t.Run("get data", func(t *testing.T) {
		assert := assert.New(t)
		ms.client.Add(key, data)
		buf, err := ms.Get(key)
		assert.Nil(err)
		assert.Empty(buf, "get invalid data should be empty")
	})
}

func TestMemoryStoreFlush(t *testing.T) {
	assert := assert.New(t)
	ttl := 10 * time.Second
	key := "a"
	value := []byte("abcd")
	config := MemoryStoreConfig{
		Size:     10,
		SaveAs:   "/tmp/elton-session-store",
		Interval: time.Second,
	}
	store, err := NewMemoryStoreByConfig(config)
	assert.Nil(err, "new memory store fail")
	_ = store.Set(key, value, ttl)
	time.Sleep(1100 * time.Millisecond)
	store.StopFlush()
	store, err = NewMemoryStoreByConfig(config)
	assert.Nil(err, "new memory store fail")
	data, err := store.Get(key)
	assert.Nil(err)
	assert.Equal(data, value, "load store from memory fail")
}
