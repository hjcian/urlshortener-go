package redis

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"goshorturl/cache/cacher"
	"goshorturl/repository"
	"time"

	redigo "github.com/gomodule/redigo/redis"
)

const (
	defaultSETEXTimeout = 30 * time.Second // to avoid deadlock
	setexKey            = "setex:%s"
)

type serializable struct {
	Url    string
	Errmsg string
}

func entry2serializable(entry *cacher.Entry) serializable {
	if entry.Err != nil {
		return serializable{entry.Url, entry.Err.Error()}
	}
	return serializable{entry.Url, ""}
}

func serialized2entry(value serializable) cacher.Entry {
	if value.Errmsg != "" {
		err := errors.New(value.Errmsg)
		if value.Errmsg == repository.ErrRecordNotFound.Error() {
			err = repository.ErrRecordNotFound
		}
		return cacher.Entry{Url: value.Url, Err: err}
	}
	return cacher.Entry{Url: value.Url, Err: nil}
}

func serialize(entry *cacher.Entry) (*bytes.Buffer, error) {
	var buffer bytes.Buffer
	s := entry2serializable(entry)
	err := gob.NewEncoder(&buffer).Encode(s)
	return &buffer, err
}

func deserialize(valBytes []byte) (*cacher.Entry, error) {
	var s serializable
	err := gob.NewDecoder(bytes.NewReader(valBytes)).Decode(&s)
	entry := serialized2entry(s)
	return &entry, err
}

type redis struct {
	pool *redigo.Pool
}

func New(host string, port int) cacher.Engine {
	pool := &redigo.Pool{
		Dial: func() (redigo.Conn, error) {
			c, err := redigo.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
			if err != nil {
				return nil, err
			}
			// NOTE: consider a basic authentication by password
			// if config.Password != "" {
			// 	if _, err := c.Do("AUTH", config.Password); err != nil {
			// 		c.Close()
			// 		return nil, err
			// 	}
			// }
			// NOTE: use number 0 as default
			// if _, err := c.Do("SELECT", 0); err != nil {
			// 	c.Close()
			// 	return nil, err
			// }
			return c, nil
		},

		// Periodic check
		TestOnBorrow: func(c redigo.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
	return &redis{pool}
}

func (r *redis) Get(id string) (*cacher.Entry, bool, error) {
	reply, err := r.do("GET", id)
	if reply == nil && err == nil {
		return nil, false, cacher.ErrEntryNotFound
	}
	if err != nil {
		return nil, false, err
	}

	data, err := redigo.Bytes(reply, err)
	if err != nil {
		return nil, false, err
	}
	entry, err := deserialize(data)
	if err != nil {
		return nil, false, err
	}
	return entry, true, nil
}

func (r *redis) Set(id string, entry *cacher.Entry, expiration time.Duration) error {
	buffer, err := serialize(entry)
	if err != nil {
		return fmt.Errorf("serialize: %w", err)
	}
	if _, err := r.do("SET", id, buffer.Bytes(), "EX", uint64(expiration.Seconds())); err != nil {
		return fmt.Errorf("call SET: %w", err)
	}
	return nil
}

func (r *redis) Delete(id string) error {
	reply, err := r.do("DEL", id)
	if err != nil {
		return err
	}
	ok, err := redigo.Bool(reply, err)
	if err != nil {
		return err
	}
	if !ok {
		return cacher.ErrEntryNotFound
	}
	return nil
}

func (r *redis) Check(id string) (bool, error) {
	script := `
local ok = redis.call('SETNX', KEYS[1], 1)
if ok == 0 then
	return 0
end

ok = redis.call('EXPIRE', KEYS[1], ARGV[1])
if ok == 0 then
	return -1
end

return ok
`
	keys := []interface{}{fmt.Sprintf(setexKey, id)}
	args := []interface{}{uint64(defaultSETEXTimeout.Seconds())}
	reply, err := r.lua(script, keys, args)

	if err != nil {
		return false, err
	}
	ret, err := redigo.Int(reply, err)
	if err != nil {
		return false, err
	}

	switch ret {
	case 1:
		return true, nil
	case 0:
		return false, nil
	default:
		return false, cacher.ErrUnexpectedError
	}
}

func (r *redis) Uncheck(id string) error {
	reply, err := r.do("DEL", fmt.Sprintf(setexKey, id))
	if err != nil {
		return err
	}
	ok, err := redigo.Bool(reply, err)
	if err != nil {
		return err
	}
	if !ok {
		return cacher.ErrEntryNotFound
	}
	return nil
}

func (r *redis) do(commandName string, args ...interface{}) (reply interface{}, err error) {
	c := r.pool.Get()
	defer c.Close()
	return c.Do(commandName, args...)
}

func (r *redis) lua(script string, keys []interface{}, args []interface{}) (interface{}, error) {
	c := r.pool.Get()
	defer c.Close()
	lua := redigo.NewScript(len(keys), script)
	return lua.Do(c, append(keys, args...)...)
}
