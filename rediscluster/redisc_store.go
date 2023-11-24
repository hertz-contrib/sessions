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
 * Copyright (c) 2012 Rodrigo Moraes. All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are
 * met:
 *
 * 	 * Redistributions of source code must retain the above copyright
 * notice, this list of conditions and the following disclaimer.
 * 	 * Redistributions in binary form must reproduce the above
 * copyright notice, this list of conditions and the following disclaimer
 * in the documentation and/or other materials provided with the
 * distribution.
 * 	 * Neither the name of Google Inc. nor the names of its
 * contributors may be used to endorse or promote products derived from
 * this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 * "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
 * A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 * OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 * SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 * LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
* This file may have been modified by CloudWeGo authors. All CloudWeGo
* Modifications are Copyright 2022 CloudWeGo Authors.
*/

package rediscluster

import (
	"context"
	"encoding/base32"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	hs "github.com/hertz-contrib/sessions"
	"github.com/redis/go-redis/v9"
)

var sessionExpire = 86400 * 30

type Store struct {
	Rdb           *redis.ClusterClient
	Codecs        []securecookie.Codec
	Opts          *sessions.Options // default configuration
	DefaultMaxAge int               // default Redis TTL for a MaxAge == 0 session
	maxLength     int
	keyPrefix     string
	serializer    hs.Serializer
}

func (s *Store) Options(options hs.Options) {
	s.Opts = options.ToGorillaOptions()
}

// NewStoreWithOption returns a new rediscluster.Store by setting *redis.ClusterOptions
func NewStoreWithOption(opt *redis.ClusterOptions, kvs ...[]byte) (*Store, error) {
	rs := &Store{
		Rdb:    redis.NewClusterClient(opt),
		Codecs: securecookie.CodecsFromPairs(kvs...),
		Opts: &sessions.Options{
			Path:   "/",
			MaxAge: sessionExpire,
		},
		DefaultMaxAge: 60 * 20, // 20 minutes seems like a reasonable default
		maxLength:     4096,
		keyPrefix:     "session_",
		serializer:    hs.GobSerializer{},
	}
	err := rs.Rdb.ForEachShard(context.Background(), func(ctx context.Context, shard *redis.Client) error {
		return shard.Ping(ctx).Err()
	})
	return rs, err
}

// NewStore returns a new rediscluster.Store
func NewStore(maxIdle int, addrs []string, password string, newClient func(opt *redis.Options) *redis.Client, kvs ...[]byte) (*Store, error) {
	return NewStoreWithOption(newOption(addrs, password, maxIdle, newClient), kvs...)
}

func (s *Store) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

func (s *Store) New(r *http.Request, name string) (*sessions.Session, error) {
	var (
		err error
		ok  bool
	)
	session := sessions.NewSession(s, name)
	// make a copy
	options := *s.Opts
	session.Options = &options
	session.IsNew = true
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...)
		if err == nil {
			ok, err = s.load(session)
			session.IsNew = !(err == nil && ok) // not new if no error and data available
		}
	}
	return session, err
}

func (s *Store) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	// Marked for deletion.
	if session.Options.MaxAge <= 0 {
		if err := s.delete(session); err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
	} else {
		// Build an alphanumeric key for the redis store.
		if session.ID == "" {
			session.ID = strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)), "=")
		}
		if err := s.save(session); err != nil {
			return err
		}
		encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
		if err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))
	}
	return nil
}

func (s *Store) Close() error {
	return s.Rdb.Close()
}

// SetMaxLength sets RedisClusterStore.maxLength if the `l` argument is greater or equal 0
// maxLength restricts the maximum length of new sessions to l.
// If l is 0 there is no limit to the size of a session, use with caution.
// The default for a new RedisClusterStore is 4096. Redis allows for max.
// value sizes of up to 512MB (http://redis.io/topics/data-types)
// Default: 4096,
func (s *Store) SetMaxLength(l int) {
	if l >= 0 {
		s.maxLength = l
	}
}

// SetKeyPrefix set the prefix
func (s *Store) SetKeyPrefix(p string) {
	s.keyPrefix = p
}

// SetSerializer sets the serializer
func (s *Store) SetSerializer(ss hs.Serializer) {
	s.serializer = ss
}

func (s *Store) load(session *sessions.Session) (bool, error) {
	res := s.Rdb.Get(context.Background(), s.keyPrefix+session.ID)
	if res == nil {
		return false, nil
	}
	b, err := res.Bytes()
	if err != nil {
		return false, err
	}
	return true, s.serializer.Deserialize(b, session)
}

// save stores the session in redis.
func (s *Store) save(session *sessions.Session) error {
	b, err := s.serializer.Serialize(session)
	if err != nil {
		return err
	}
	if s.maxLength != 0 && len(b) > s.maxLength {
		return errors.New("SessionStore: the value to store is too big")
	}
	age := session.Options.MaxAge
	if age == 0 {
		age = s.DefaultMaxAge
	}
	err = s.Rdb.SetEx(context.Background(), s.keyPrefix+session.ID, b, time.Duration(age)*time.Second).Err()
	return err
}

func (s *Store) ping() (bool, error) {
	res := s.Rdb.Ping(context.Background())
	if result, err := res.Result(); result != "PONG" || err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) delete(session *sessions.Session) error {
	del := s.Rdb.Del(context.Background(), s.keyPrefix+session.ID)
	return del.Err()
}

// LoadSessionBySessionId Get session using session_id even without a context
func LoadSessionBySessionId(s *Store, sessionId string) (*sessions.Session, error) {
	var session sessions.Session
	session.ID = sessionId
	exist, err := s.load(&session)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return &session, nil
}

// SaveSessionWithoutContext Save session even without a context
func SaveSessionWithoutContext(s *Store, sessionId string, session *sessions.Session) error {
	session.ID = sessionId
	return s.save(session)
}

func newOption(
	addrs []string,
	password string,
	maxIdleConns int,
	newClient func(opt *redis.Options) *redis.Client,
) *redis.ClusterOptions {
	return &redis.ClusterOptions{
		Addrs:        addrs,
		NewClient:    newClient,
		Password:     password,
		MaxIdleConns: maxIdleConns,
	}
}
