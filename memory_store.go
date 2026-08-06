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
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/vicanso/hes"
)

const defaultFlushInterval = 60 * time.Second

// ErrNotInit is returned when MemoryStore methods are called before initialization.
var ErrNotInit = hes.New(
	"client not init",
	hes.WithCategory(ErrCategory),
	hes.WithStatus(http.StatusInternalServerError),
	hes.WithException(),
)

// MemoryStore is an in-process LRU session store with optional file persistence.
type MemoryStore struct {
	client *lru.Cache[string, *MemoryStoreInfo]

	stopOnce sync.Once
	stopCh   chan struct{}
}

// MemoryStoreInfo is a single cached session entry.
type MemoryStoreInfo struct {
	ExpiredAt int64  `json:"expiredAt"`
	Data      []byte `json:"data"`
}

// MemoryStoreConfig configures NewMemoryStoreByConfig.
type MemoryStoreConfig struct {
	// Size is the max number of sessions retained by the LRU.
	Size int
	// SaveAs is an optional file path for periodic snapshot persistence.
	SaveAs string
	// Interval is the flush period when SaveAs is set. Defaults to 60s;
	// values below 1s are clamped to the default.
	Interval time.Duration
}

// Get returns session data from memory. Expired entries are treated as missing.
func (ms *MemoryStore) Get(_ context.Context, key string) ([]byte, error) {
	if ms.client == nil {
		return nil, ErrNotInit
	}
	info, found := ms.client.Get(key)
	if !found {
		return nil, nil
	}
	if info.ExpiredAt < time.Now().Unix() {
		return nil, nil
	}
	return info.Data, nil
}

// Set stores session data in memory with a TTL.
func (ms *MemoryStore) Set(_ context.Context, key string, data []byte, ttl time.Duration) error {
	if ms.client == nil {
		return ErrNotInit
	}
	ms.client.Add(key, &MemoryStoreInfo{
		ExpiredAt: time.Now().Unix() + int64(ttl.Seconds()),
		Data:      data,
	})
	return nil
}

// Destroy removes a session from memory.
func (ms *MemoryStore) Destroy(_ context.Context, key string) error {
	if ms.client == nil {
		return ErrNotInit
	}
	ms.client.Remove(key)
	return nil
}

func (ms *MemoryStore) snapshot() map[string]*MemoryStoreInfo {
	keys := ms.client.Keys()
	now := time.Now().Unix()
	m := make(map[string]*MemoryStoreInfo, len(keys))
	for _, key := range keys {
		info, found := ms.client.Get(key)
		if !found || info.ExpiredAt < now {
			continue
		}
		m[key] = info
	}
	return m
}

func (ms *MemoryStore) flushLoop(saveAs string, interval time.Duration) {
	if interval < time.Second {
		interval = defaultFlushInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ms.stopCh:
			return
		case <-ticker.C:
			buf, err := json.Marshal(ms.snapshot())
			if err != nil {
				continue
			}
			_ = os.WriteFile(saveAs, buf, 0o600)
		}
	}
}

// StopFlush stops the background file flush goroutine (idempotent).
func (ms *MemoryStore) StopFlush() {
	if ms.stopCh == nil {
		return
	}
	ms.stopOnce.Do(func() {
		close(ms.stopCh)
	})
}

// NewMemoryStore creates an in-memory store with the given LRU capacity.
func NewMemoryStore(size int) (*MemoryStore, error) {
	client, err := lru.New[string, *MemoryStoreInfo](size)
	if err != nil {
		return nil, err
	}
	return &MemoryStore{client: client}, nil
}

// NewMemoryStoreByConfig creates a memory store and optionally restores from /
// periodically snapshots to a file when SaveAs is set.
func NewMemoryStoreByConfig(config MemoryStoreConfig) (*MemoryStore, error) {
	store, err := NewMemoryStore(config.Size)
	if err != nil {
		return nil, err
	}
	if config.SaveAs == "" {
		return store, nil
	}

	if buf, err := os.ReadFile(config.SaveAs); err == nil && len(buf) > 0 {
		m := make(map[string]*MemoryStoreInfo)
		if json.Unmarshal(buf, &m) == nil {
			for key, value := range m {
				store.client.Add(key, value)
			}
		}
	}

	store.stopCh = make(chan struct{})
	go store.flushLoop(config.SaveAs, config.Interval)
	return store, nil
}
