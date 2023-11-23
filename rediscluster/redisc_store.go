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
	"encoding/base32"
	"errors"
	"net/http"
	"strings"
	"time"

	hs "github.com/hertz-contrib/sessions"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	rredis "github.com/hertz-contrib/sessions/redis"
	"github.com/mna/redisc"
)

var sessionExpire = 86400 * 30

type Store struct {
	Cluster       *redisc.Cluster
	Codecs        []securecookie.Codec
	Opts          *sessions.Options // default configuration
	DefaultMaxAge int               // default Redis TTL for a MaxAge == 0 session
	maxLength     int
	keyPrefix     string
	serializer    rredis.SessionSerializer
}

func (s *Store) Options(options hs.Options) {
	s.Opts = options.ToGorillaOptions()
}

// NewStoreWithCluster returns a new rediscluster.Store by setting redisc.Cluster
func NewStoreWithCluster(cluster *redisc.Cluster, kvs ...[]byte) (*Store, error) {
	rs := &Store{
		Cluster: cluster,
		Codecs:  securecookie.CodecsFromPairs(kvs...),
		Opts: &sessions.Options{
			Path:   "/",
			MaxAge: sessionExpire,
		},
		DefaultMaxAge: 60 * 20, // 20 minutes seems like a reasonable default
		maxLength:     4096,
		keyPrefix:     "session_",
		serializer:    rredis.GobSerializer{},
	}

	err := cluster.Refresh()
	return rs, err
}

// NewStore returns a new rediscluster.Store
func NewStore(maxIdle int, network string, startupNodes []string, password string, kvs ...[]byte) (*Store, error) {
	return NewStoreWithCluster(newCluster(startupNodes, CreateDefaultPool(maxIdle, network, password)), kvs...)
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
	return s.Cluster.Close()
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
func (s *Store) SetSerializer(ss rredis.SessionSerializer) {
	s.serializer = ss
}

func (s *Store) load(session *sessions.Session) (bool, error) {
	conn := s.Cluster.Get()
	defer conn.Close()
	if err := conn.Err(); err != nil {
		return false, err
	}
	data, err := conn.Do("GET", s.keyPrefix+session.ID)
	if err != nil {
		return false, err
	}
	if data == nil {
		return false, nil // no data was associated with this key
	}
	b, err := redis.Bytes(data, err)
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
	conn := s.Cluster.Get()
	defer conn.Close()
	if err = conn.Err(); err != nil {
		return err
	}
	age := session.Options.MaxAge
	if age == 0 {
		age = s.DefaultMaxAge
	}
	_, err = conn.Do("SETEX", s.keyPrefix+session.ID, age, b)
	return err
}

func (s *Store) ping() (bool, error) {
	conn := s.Cluster.Get()
	defer conn.Close()
	data, err := conn.Do("PING")
	if err != nil || data == nil {
		return false, err
	}
	return data == "PONG", nil
}

func (s *Store) delete(session *sessions.Session) error {
	conn := s.Cluster.Get()
	defer conn.Close()
	if _, err := conn.Do("DEL", s.keyPrefix+session.ID); err != nil {
		return err
	}
	return nil
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

// CreateDefaultPool return a function is used to set redisc.Cluster CreatePool by default
func CreateDefaultPool(maxIdle int, network string, password ...string) func(address string, options ...redis.DialOption) (*redis.Pool, error) {
	return func(address string, options ...redis.DialOption) (*redis.Pool, error) {
		var pwd string
		if len(password) > 0 {
			pwd = password[0]
		}
		pool := &redis.Pool{
			MaxIdle:     maxIdle,
			IdleTimeout: 240 * time.Second,
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				return err
			},
			Dial: func() (redis.Conn, error) {
				return dial(network, address, pwd)
			},
		}
		conn := pool.Get()
		defer conn.Close()
		data, err := conn.Do("PING")
		if err != nil || data == nil {
			return nil, err
		}
		return pool, nil
	}
}

func newCluster(
	startupNodes []string,
	createPoolFunc func(address string, options ...redis.DialOption) (*redis.Pool, error),
) *redisc.Cluster {
	return &redisc.Cluster{
		StartupNodes: startupNodes,
		CreatePool:   createPoolFunc,
	}
}

func dial(network, address, password string) (redis.Conn, error) {
	c, err := redis.Dial(network, address)
	if err != nil {
		return nil, err
	}
	if password != "" {
		if _, err := c.Do("AUTH", password); err != nil {
			c.Close()
			return nil, err
		}
	}
	return c, err
}
