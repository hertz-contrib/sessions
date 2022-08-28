// The MIT License (MIT)
//
// Copyright (c) 2016 Bo-Yi Wu
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// This file may have been modified by CloudWeGo authors. All CloudWeGo
// Modifications are Copyright 2022 CloudWeGo Authors.

package redis

import (
	"errors"

	"github.com/boj/redistore"
	"github.com/gomodule/redigo/redis"
	"github.com/hertz-contrib/sessions"
)

type Store interface {
	sessions.Store
}

type store struct {
	*redistore.RediStore
}

func (s *store) Options(opts sessions.Options) {
	s.RediStore.Options = opts.ToGorillaOptions()
}

func NewStore(size int, network, addr, passwd string, keyPairs ...[]byte) (Store, error) {
	s, err := redistore.NewRediStore(size, network, addr, passwd, keyPairs...)
	if err != nil {
		return nil, err
	}
	return &store{s}, nil
}

// NewStoreWithDB - like NewStore but accepts `DB` parameter to select
// redis DB instead of using the default one ("0")
//
// Ref: https://godoc.org/github.com/boj/redistore#NewRediStoreWithDB
func NewStoreWithDB(size int, network, addr, passwd, DB string, keyPairs ...[]byte) (Store, error) {
	s, err := redistore.NewRediStoreWithDB(size, network, addr, passwd, DB, keyPairs...)
	if err != nil {
		return nil, err
	}
	return &store{s}, nil
}

// NewStoreWithPool instantiates a RediStore with a *redis.Pool passed in.
//
// Ref: https://godoc.org/github.com/boj/redistore#NewRediStoreWithPool
func NewStoreWithPool(pool *redis.Pool, keyPairs ...[]byte) (Store, error) {
	s, err := redistore.NewRediStoreWithPool(pool, keyPairs...)
	if err != nil {
		return nil, err
	}
	return &store{s}, nil
}

// SetKeyPrefix sets the key prefix in the redis database.
func SetKeyPrefix(s Store, prefix string) error {
	rediStore, err := GetRedisStore(s)
	if err != nil {
		return err
	}

	rediStore.SetKeyPrefix(prefix)
	return nil
}

// GetRedisStore get the actual woking store.
// Ref: https://godoc.org/github.com/boj/redistore#RediStore
func GetRedisStore(s Store) (rediStore *redistore.RediStore, err error) {
	realStore, ok := s.(*store)
	if !ok {
		err = errors.New("unable to get the redis store: Store isn't *store")
		return
	}

	rediStore = realStore.RediStore
	return
}
