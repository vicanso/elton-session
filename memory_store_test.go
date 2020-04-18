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
