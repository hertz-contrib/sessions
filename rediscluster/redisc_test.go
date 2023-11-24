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
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/gorilla/sessions"
)

func init() {
	gob.Register(FlashMessage{})
}

// ----------------------------------------------------------------------------
// ResponseRecorder
// ----------------------------------------------------------------------------
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// ResponseRecorder is an implementation of http.ResponseWriter that
// records its mutations for later inspection in tests.
type ResponseRecorder struct {
	Code      int           // the HTTP response code from WriteHeader
	HeaderMap http.Header   // the HTTP response headers
	Body      *bytes.Buffer // if non-nil, the bytes.Buffer to append written data to
	Flushed   bool
}

// NewRecorder returns an initialized ResponseRecorder.
func NewRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		HeaderMap: make(http.Header),
		Body:      new(bytes.Buffer),
	}
}

// Header returns the response headers.
func (rw *ResponseRecorder) Header() http.Header {
	return rw.HeaderMap
}

// Write always succeeds and writes to rw.Body, if not nil.
func (rw *ResponseRecorder) Write(buf []byte) (int, error) {
	if rw.Body != nil {
		rw.Body.Write(buf)
	}
	if rw.Code == 0 {
		rw.Code = http.StatusOK
	}
	return len(buf), nil
}

// WriteHeader sets rw.Code.
func (rw *ResponseRecorder) WriteHeader(code int) {
	rw.Code = code
}

// Flush sets rw.Flushed to true.
func (rw *ResponseRecorder) Flush() {
	rw.Flushed = true
}

// ----------------------------------------------------------------------------

type FlashMessage struct {
	Type    int
	Message string
}

func TestNewRedisCluster(t *testing.T) {
	var (
		req     *http.Request
		rsp     *ResponseRecorder
		hdr     http.Header
		ok      bool
		cookies []string
		session *sessions.Session
		flashes []interface{}
	)

	{
		store, err := NewStore(10, []string{"localhost:5000", "localhost:5001"}, "", nil, []byte("secret-key"))

		assert.Nil(t, err)
		defer store.Close()

		req, _ = http.NewRequest("GET", "http://localhost:8080/", nil)
		rsp = NewRecorder()
		// Get a session.
		if session, err = store.Get(req, "session-key"); err != nil {
			t.Fatalf("Error getting session: %v", err)
		}
		// Get a flash.
		flashes = session.Flashes()
		if len(flashes) != 0 {
			t.Errorf("Expected empty flashes; Got %v", flashes)
		}
		// Add some flashes.
		session.AddFlash("foo")
		session.AddFlash("bar")
		// Custom key.
		session.AddFlash("baz", "custom_key")
		// Save.
		if err = sessions.Save(req, rsp); err != nil {
			t.Fatalf("Error saving session: %v", err)
		}
		hdr = rsp.Header()
		cookies, ok = hdr["Set-Cookie"]
		if !ok || len(cookies) != 1 {
			t.Fatalf("No cookies. Header: %s", hdr)
		}
	}

	{
		store, err := NewStore(10, []string{"localhost:5000", "localhost:5001"}, "", nil, []byte("secret-key"))
		if err != nil {
			t.Fatal(err.Error())
		}
		defer store.Close()

		req, _ := http.NewRequest("GET", "http://localhost:8080/", nil)
		req.Header.Add("Cookie", cookies[0])
		rsp = NewRecorder()
		// Get a session.
		if session, err = store.Get(req, "session-key"); err != nil {
			t.Fatalf("Error getting session: %v", err)
		}
		// Check all saved values.
		flashes = session.Flashes()
		if len(flashes) != 2 {
			t.Fatalf("Expected flashes; Got %v", flashes)
		}
		if flashes[0] != "foo" || flashes[1] != "bar" {
			t.Errorf("Expected foo,bar; Got %v", flashes)
		}
		flashes = session.Flashes()
		if len(flashes) != 0 {
			t.Errorf("Expected dumped flashes; Got %v", flashes)
		}
		// Custom key.
		flashes = session.Flashes("custom_key")
		if len(flashes) != 1 {
			t.Errorf("Expected flashes; Got %v", flashes)
		} else if flashes[0] != "baz" {
			t.Errorf("Expected baz; Got %v", flashes)
		}
		flashes = session.Flashes("custom_key")
		if len(flashes) != 0 {
			t.Errorf("Expected dumped flashes; Got %v", flashes)
		}

		// RediStore specific
		// Set MaxAge to -1 to mark for deletion.
		session.Options.MaxAge = -1
		// Save.
		if err = sessions.Save(req, rsp); err != nil {
			t.Fatalf("Error saving session: %v", err)
		}
	}

	{
		store, err := NewStore(10, []string{"localhost:5000", "localhost:5001"}, "", nil, []byte("secret-key"))
		if err != nil {
			t.Fatal(err.Error())
		}
		defer store.Close()
		req, _ = http.NewRequest("GET", "http://localhost:8080/", nil)
		rsp = NewRecorder()
		// Get a session.
		if session, err = store.Get(req, "session-key"); err != nil {
			t.Fatalf("Error getting session: %v", err)
		}
		// Get a flash.
		flashes = session.Flashes()
		if len(flashes) != 0 {
			t.Errorf("Expected empty flashes; Got %v", flashes)
		}
		// Add some flashes.
		session.AddFlash(&FlashMessage{42, "foo"})
		// Save.
		if err = sessions.Save(req, rsp); err != nil {
			t.Fatalf("Error saving session: %v", err)
		}
		hdr = rsp.Header()
		cookies, ok = hdr["Set-Cookie"]
		if !ok || len(cookies) != 1 {
			t.Fatalf("No cookies. Header: %s", hdr)
		}

		sID := session.ID
		s, err := LoadSessionBySessionId(store, sID)
		if err != nil {
			t.Fatalf("LoadSessionBySessionId Error: %v", err)
		}

		if len(s.Values) == 0 {
			t.Fatalf("No session value: %s", hdr)
		}
	}

	{
		store, err := NewStore(10, []string{"localhost:5000", "localhost:5001"}, "", nil, []byte("secret-key"))
		if err != nil {
			t.Fatal(err.Error())
		}
		defer store.Close()

		req, _ := http.NewRequest("GET", "http://localhost:8080/", nil)
		req.Header.Add("Cookie", cookies[0])
		rsp = NewRecorder()
		// Get a session.
		if session, err = store.Get(req, "session-key"); err != nil {
			t.Fatalf("Error getting session: %v", err)
		}
		// Check all saved values.
		flashes = session.Flashes()
		if len(flashes) != 1 {
			t.Fatalf("Expected flashes; Got %v", flashes)
		}
		custom := flashes[0].(FlashMessage)
		if custom.Type != 42 || custom.Message != "foo" {
			t.Errorf("Expected %#v, got %#v", FlashMessage{42, "foo"}, custom)
		}

		// RediStore specific
		// Set MaxAge to -1 to mark for deletion.
		session.Options.MaxAge = -1
		// Save.
		if err = sessions.Save(req, rsp); err != nil {
			t.Fatalf("Error saving session: %v", err)
		}
	}

	{
		store, err := NewStore(10, []string{"localhost:5000", "localhost:5001"}, "", nil, []byte("secret-key"))
		if err != nil {
			t.Fatal(err.Error())
		}
		defer store.Close()

		req, err = http.NewRequest("GET", "http://www.example.com", nil)
		if err != nil {
			t.Fatal("failed to create request", err)
		}
		w := httptest.NewRecorder()

		session, err = store.New(req, "my session")
		if err != nil {
			t.Fatal("failed to New store", err)
		}
		session.Values["big"] = make([]byte, base64.StdEncoding.DecodedLen(4096*2))
		err = session.Save(req, w)
		if err == nil {
			t.Fatal("expected an error, got nil")
		}

		store.SetMaxLength(4096 * 3) // A bit more than the value size to account for encoding overhead.
		err = session.Save(req, w)
		if err != nil {
			t.Fatal("failed to Save:", err)
		}
	}

	{
		store, err := NewStore(10, []string{"localhost:5000", "localhost:5001"}, "", nil, []byte("secret-key"))
		if err != nil {
			t.Fatal(err.Error())
		}
		defer store.Close()

		req, _ = http.NewRequest("GET", "http://localhost:8080/", nil)
		rsp = NewRecorder()
		// Get a session.
		if session, err = store.Get(req, "session-key"); err != nil {
			t.Fatalf("Error getting session: %v", err)
		}
		// Get a flash.
		flashes = session.Flashes()
		if len(flashes) != 0 {
			t.Errorf("Expected empty flashes; Got %v", flashes)
		}
		// Add some flashes.
		session.AddFlash("foo")
		// Save.
		if err = sessions.Save(req, rsp); err != nil {
			t.Fatalf("Error saving session: %v", err)
		}
		hdr = rsp.Header()
		cookies, ok = hdr["Set-Cookie"]
		if !ok || len(cookies) != 1 {
			t.Fatalf("No cookies. Header: %s", hdr)
		}

		// Get a session.
		req.Header.Add("Cookie", cookies[0])
		if session, err = store.Get(req, "session-key"); err != nil {
			t.Fatalf("Error getting session: %v", err)
		}
		// Check all saved values.
		flashes = session.Flashes()
		if len(flashes) != 1 {
			t.Fatalf("Expected flashes; Got %v", flashes)
		}
		if flashes[0] != "foo" {
			t.Errorf("Expected foo,bar; Got %v", flashes)
		}
	}
}

func TestPingGoodPort(t *testing.T) {
	store, err := NewStore(10, []string{"localhost:5000", "localhost:5001"}, "", nil, []byte("secret-key"))
	defer store.Close()
	ok, err := store.ping()
	if err != nil {
		t.Error(err.Error())
	}
	if !ok {
		t.Error("Expected server to PONG")
	}
}
