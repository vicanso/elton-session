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
	"errors"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

var (
	// ErrNotInit error not init
	ErrNotInit = errors.New("client not init")
)

type (
	// MemoryStore memory store for session
	MemoryStore struct {
		client *lru.Cache
	}
	// MemoryStoreInfo memory store info
	MemoryStoreInfo struct {
		ExpiredAt int64
		Data      []byte
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
