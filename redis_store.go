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

package codsession

import (
	"errors"
	"time"

	"github.com/go-redis/redis"
)

type (
	// RedisStore redis store for session
	RedisStore struct {
		client *redis.Client
	}
)

// Get get the session from redis
func (rs *RedisStore) Get(key string) ([]byte, error) {
	buf, err := rs.client.Get(key).Bytes()
	if err == redis.Nil {
		return buf, nil
	}
	return buf, err
}

// Set set the session to redis
func (rs *RedisStore) Set(key string, data []byte, ttl time.Duration) error {
	return rs.client.Set(key, data, ttl).Err()
}

// Destroy remove the session from redis
func (rs *RedisStore) Destroy(key string) error {
	return rs.client.Del(key).Err()
}

// NewRedisStore create new redis store instance
func NewRedisStore(client *redis.Client, opts *redis.Options) *RedisStore {
	if client == nil && opts == nil {
		panic(errors.New("client and opts can both be nil"))
	}
	rs := &RedisStore{}
	if client != nil {
		rs.client = client
	} else {
		rs.client = redis.NewClient(opts)
	}
	return rs
}
