// Copyright 2019 tree xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		err = json.Unmarshal(buf, &m)
		if err != nil {
			return
		}
		for key, value := range m {
			store.client.Add(key, value)
		}
		// 定时写入文件
		go store.intervalFlush(file, config.Interval)
	}
	return
}
