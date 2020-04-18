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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/vicanso/hes"
)

var (
	// ErrNotInit error not init
	ErrNotInit = &hes.Error{
		Message:    "client not init",
		Category:   ErrCategory,
		StatusCode: http.StatusInternalServerError,
		Exception:  true,
	}
	defaultInterval = 60 * time.Second
)

const (
	flushStatusStop = iota
	flushStatusRunning
)

type (
	// MemoryStore memory store for session
	MemoryStore struct {
		client      *lru.Cache
		flushStatus int32
	}
	// MemoryStoreInfo memory store info
	MemoryStoreInfo struct {
		ExpiredAt int64
		Data      []byte
	}
	// MemoryStoreConfig memory store config
	MemoryStoreConfig struct {
		Size int
		// SaveAs save as file
		SaveAs string
		// Interval save interval
		Interval time.Duration
	}
)

// Get get the seesion from memory
func (ms *MemoryStore) Get(key string) (data []byte, err error) {
	client := ms.client
	if client == nil {
		err = ErrNotInit
		return
	}
	v, found := client.Get(key)
	if !found {
		return
	}
	info, ok := v.(*MemoryStoreInfo)
	if !ok {
		return
	}
	if info.ExpiredAt < time.Now().Unix() {
		return
	}
	data = info.Data
	return
}

// Set set the session to memory
func (ms *MemoryStore) Set(key string, data []byte, ttl time.Duration) (err error) {
	client := ms.client
	if client == nil {
		err = ErrNotInit
		return
	}
	expiredAt := time.Now().Unix() + int64(ttl.Seconds())
	info := &MemoryStoreInfo{
		ExpiredAt: expiredAt,
		Data:      data,
	}
	client.Add(key, info)
	return
}

// Destroy remove the session from memory
func (ms *MemoryStore) Destroy(key string) (err error) {
	client := ms.client
	if client == nil {
		err = ErrNotInit
		return
	}
	client.Remove(key)
	return
}

func (ms *MemoryStore) intervalFlush(saveAs string, interval time.Duration) {
	client := ms.client
	if client == nil {
		return
	}
	atomic.StoreInt32(&ms.flushStatus, flushStatusRunning)
	if interval < time.Second {
		interval = defaultInterval
	}
	ticker := time.NewTicker(interval)
	for range ticker.C {
		if atomic.LoadInt32(&ms.flushStatus) == flushStatusStop {
			return
		}
		keys := client.Keys()
		m := make(map[string]*MemoryStoreInfo)
		for _, k := range keys {
			key, _ := k.(string)
			if key == "" {
				continue
			}
			v, found := client.Get(key)
			if !found {
				continue
			}
			info, ok := v.(*MemoryStoreInfo)
			if !ok {
				continue
			}
			if info.ExpiredAt < time.Now().Unix() {
				continue
			}
			m[key] = info
		}

		buf, _ := json.Marshal(&m)
		_ = ioutil.WriteFile(saveAs, buf, 0600)
	}
}

// StopFlush stop flush
func (ms *MemoryStore) StopFlush() {
	atomic.StoreInt32(&ms.flushStatus, flushStatusStop)
}

// NewMemoryStore create new memory store instance
func NewMemoryStore(size int) (store *MemoryStore, err error) {
	client, err := lru.New(size)
	if err != nil {
		return
	}
	store = &MemoryStore{
		client: client,
	}
	return
}

// NewMemoryStoreByConfig create new memory store instance by config
func NewMemoryStoreByConfig(config MemoryStoreConfig) (store *MemoryStore, err error) {
	store, err = NewMemoryStore(config.Size)
	if err != nil {
		return
	}
	file := config.SaveAs
	if file != "" {
		// 从文件中恢复
		buf, _ := ioutil.ReadFile(file)
		m := make(map[string]*MemoryStoreInfo)
		// 如果读取失败，则忽略
		_ = json.Unmarshal(buf, &m)
		for key, value := range m {
			store.client.Add(key, value)
		}
		// 定时写入文件
		go store.intervalFlush(file, config.Interval)
	}
	return
}
