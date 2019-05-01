package session

import (
	"bytes"
	"testing"
	"time"
)

func TestMemoryStore(t *testing.T) {
	key := generateID()
	data := []byte("tree.xie")
	ttl := 300 * time.Second
	ms, _ := NewMemoryStore(1024)

	t.Run("not init", func(t *testing.T) {
		tmp := &MemoryStore{}
		_, err := tmp.Get(key)
		if err != ErrNotInit {
			t.Fatalf("should return not init error")
		}
		err = tmp.Set(key, data, ttl)
		if err != ErrNotInit {
			t.Fatalf("should return not init error")
		}
		err = tmp.Destroy(key)
		if err != ErrNotInit {
			t.Fatalf("should return not init error")
		}
	})

	t.Run("get not exists data", func(t *testing.T) {
		buf, err := ms.Get(key)
		if err != nil || len(buf) != 0 {
			t.Fatalf("should return empty bytes")
		}
	})

	t.Run("set data", func(t *testing.T) {
		err := ms.Set(key, data, ttl)
		if err != nil {
			t.Fatalf("set data fail, %v", err)
		}
		buf, err := ms.Get(key)
		if err != nil {
			t.Fatalf("get data fail after set, %v", err)
		}
		if !bytes.Equal(data, buf) {
			t.Fatalf("the data is not the same after set")
		}
	})

	t.Run("destroy", func(t *testing.T) {
		err := ms.Destroy(key)
		if err != nil {
			t.Fatalf("destory data fail, %v", err)
		}
		buf, err := ms.Get(key)
		if err != nil || len(buf) != 0 {
			t.Fatalf("should return empty bytes after destroy")
		}
	})

	t.Run("expired", func(t *testing.T) {
		err := ms.Set(key, data, 0)
		if err != nil {
			t.Fatalf("set data fail, %v", err)
		}
		time.Sleep(time.Second)
		buf, err := ms.Get(key)
		if err != nil {
			t.Fatalf("get data fail after set, %v", err)
		}
		if len(buf) != 0 {
			t.Fatalf("expired data should be nil")
		}
	})
}

func TestMemoryStoreFlush(t *testing.T) {
	ttl := 10 * time.Second
	key := "a"
	value := []byte("abcd")
	config := MemoryStoreConfig{
		Size:     10,
		SaveAs:   "/tmp/cod-session-store",
		Interval: time.Second,
	}
	store, err := NewMemoryStoreByConfig(config)
	if err != nil {
		t.Fatalf("new memory store fail, %v", err)
	}
	store.Set(key, value, ttl)
	time.Sleep(1100 * time.Millisecond)
	store.StopFlush()
	store, err = NewMemoryStoreByConfig(config)
	if err != nil {
		t.Fatalf("new memory store fail, %v", err)
	}
	data, err := store.Get(key)
	if err != nil {
		t.Fatalf("get session fail, %v", err)
	}
	if !bytes.Equal(data, value) {
		t.Fatalf("load store from fail")
	}
}
