package session

import (
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

func TestRedisStore(t *testing.T) {
	key := generateID()
	data := []byte("tree.xie")
	ttl := 300 * time.Second
	rs := NewRedisStore(nil, &redis.Options{
		Addr: "localhost:6379",
	})

	t.Run("client and opts both nil", func(t *testing.T) {
		assert := assert.New(t)
		done := false
		defer func() {
			r := recover()
			assert.Equal(r.(error), errClientAndOptBothNil)
			done = true
		}()
		NewRedisStore(nil, nil)
		assert.True(done)
	})

	t.Run("get key", func(t *testing.T) {
		assert := assert.New(t)
		rs := RedisStore{
			Prefix: "ss-",
		}
		assert.Equal(rs.getKey("a"), "ss-a")
	})
	t.Run("new redis store", func(t *testing.T) {
		client := redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})
		NewRedisStore(client, nil)

		NewRedisStore(nil, &redis.Options{
			Addr: "localhost:6379",
		})
	})
	t.Run("get not exists data", func(t *testing.T) {
		assert := assert.New(t)
		buf, err := rs.Get(key)
		assert.Nil(err)
		assert.Empty(buf, "should return empty bytes")
	})

	t.Run("set data", func(t *testing.T) {
		assert := assert.New(t)
		err := rs.Set(key, data, ttl)
		assert.Nil(err)
		buf, err := rs.Get(key)
		assert.Nil(err)
		assert.Equal(data, buf, "the data isn't the same after set")
	})

	t.Run("destroy", func(t *testing.T) {
		assert := assert.New(t)
		err := rs.Destroy(key)
		assert.Nil(err)
		buf, err := rs.Get(key)
		assert.Nil(err)
		assert.Empty(buf, "should return empty bytes after destroy")
	})
}
