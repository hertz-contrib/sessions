/*
 * Copyright 2023 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * The MIT License (MIT)
 *
 * Copyright (c) 2016 Bo-Yi Wu
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 *
* This file may have been modified by CloudWeGo authors. All CloudWeGo
* Modifications are Copyright 2022 CloudWeGo Authors.
*/

package redis

import (
	"errors"

	"github.com/gomodule/redigo/redis"
	"github.com/hertz-contrib/sessions"
)

type Store interface {
	sessions.Store
}

type store struct {
	*RediStore
}

func (s *store) Options(opts sessions.Options) {
	s.RediStore.Options = opts.ToGorillaOptions()
}

func NewStore(size int, network, addr, passwd string, keyPairs ...[]byte) (Store, error) {
	s, err := NewRediStore(size, network, addr, passwd, keyPairs...)
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
	s, err := NewRediStoreWithDB(size, network, addr, passwd, DB, keyPairs...)
	if err != nil {
		return nil, err
	}
	return &store{s}, nil
}

// NewStoreWithPool instantiates a RediStore with a *redis.Pool passed in.
//
// Ref: https://godoc.org/github.com/boj/redistore#NewRediStoreWithPool
func NewStoreWithPool(pool *redis.Pool, keyPairs ...[]byte) (Store, error) {
	s, err := NewRediStoreWithPool(pool, keyPairs...)
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

// GetRedisStore get the actual working store.
// Ref: https://godoc.org/github.com/boj/redistore#RediStore
func GetRedisStore(s Store) (rediStore *RediStore, err error) {
	realStore, ok := s.(*store)
	if !ok {
		err = errors.New("unable to get the redis store: Store isn't *store")
		return
	}

	rediStore = realStore.RediStore
	return
}
