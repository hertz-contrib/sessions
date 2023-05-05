//Copyright (c) 2013 Brian Jones
//
//Permission is hereby granted, free of charge, to any person obtaining a copy
//of this software and associated documentation files (the "Software"), to deal
//in the Software without restriction, including without limitation the rights
//to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//copies of the Software, and to permit persons to whom the Software is
//furnished to do so, subject to the following conditions:
//
//The above copyright notice and this permission notice shall be included in
//all copies or substantial portions of the Software.
//
//THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
//THE SOFTWARE.
//
// This file may have been modified by CloudWeGo authors. All CloudWeGo
// Modifications are Copyright 2022 CloudWeGo Authors.

package redis

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
)

func setup() string {
	return redisTestServer
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

func TestRediStore(t *testing.T) {
	var (
		req     *http.Request
		rsp     *ResponseRecorder
		hdr     http.Header
		ok      bool
		cookies []string
		session *sessions.Session
		flashes []interface{}
	)

	// Copyright 2012 The Gorilla Authors. All rights reserved.
	// Use of this source code is governed by a BSD-style
	// license that can be found in the LICENSE file.

	// Round 1 ----------------------------------------------------------------
	{
		// RedisStore
		addr := setup()
		store, err := NewRediStore(10, "tcp", addr, "", []byte("secret-key"))
		if err != nil {
			t.Fatal(err.Error())
		}
		defer store.Close()

		req, _ := http.NewRequest("GET", "http://localhost:8080/", nil)
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

	// Round 2 ----------------------------------------------------------------
	{

		addr := setup()
		store, err := NewRediStore(10, "tcp", addr, "", []byte("secret-key"))
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

	// Round 3 ----------------------------------------------------------------
	// Custom type

	// RedisStore
	{
		addr := setup()
		store, err := NewRediStore(10, "tcp", addr, "", []byte("secret-key"))
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
	}

	// Round 4 ----------------------------------------------------------------
	// Custom type
	{
		addr := setup()
		store, err := NewRediStore(10, "tcp", addr, "", []byte("secret-key"))
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

	// Round 5 ----------------------------------------------------------------
	// RediStore Delete session (deprecated)

	//req, _ = http.NewRequest("GET", "http://localhost:8080/", nil)
	//req.Header.Add("Cookie", cookies[0])
	//rsp = NewRecorder()
	//// Get a session.
	//if session, err = store.Get(req, "session-key"); err != nil {
	//	t.Fatalf("Error getting session: %v", err)
	//}
	//// Delete session.
	//if err = store.Delete(req, rsp, session); err != nil {
	//	t.Fatalf("Error deleting session: %v", err)
	//}
	//// Get a flash.
	//flashes = session.Flashes()
	//if len(flashes) != 0 {
	//	t.Errorf("Expected empty flashes; Got %v", flashes)
	//}
	//hdr = rsp.Header()
	//cookies, ok = hdr["Set-Cookie"]
	//if !ok || len(cookies) != 1 {
	//	t.Fatalf("No cookies. Header:", hdr)
	//}

	// Round 6 ----------------------------------------------------------------
	// RediStore change MaxLength of session

	{
		addr := setup()
		store, err := NewRediStore(10, "tcp", addr, "", []byte("secret-key"))
		if err != nil {
			t.Fatal(err.Error())
		}
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

	// Round 7 ----------------------------------------------------------------

	// RedisStoreWithDB
	{
		addr := setup()
		store, err := NewRediStoreWithDB(10, "tcp", addr, "", "1", []byte("secret-key"))
		if err != nil {
			t.Fatal(err.Error())
		}
		defer store.Close()

		req, _ = http.NewRequest("GET", "http://localhost:8080/", nil)
		rsp = NewRecorder()
		// Get a session. Using the same key as previously, but on different DB
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

	// Round 8 ----------------------------------------------------------------
	// JSONSerializer

	// RedisStore
	{
		addr := setup()
		store, err := NewRediStore(10, "tcp", addr, "", []byte("secret-key"))
		store.SetSerializer(JSONSerializer{})
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
	store, _ := NewRediStore(10, "tcp", ":6379", "", []byte("secret-key"))
	defer store.Close()
	ok, err := store.ping()
	if err != nil {
		t.Error(err.Error())
	}
	if !ok {
		t.Error("Expected server to PONG")
	}
}

func TestPingBadPort(t *testing.T) {
	store, _ := NewRediStore(10, "tcp", ":6378", "", []byte("secret-key"))
	defer store.Close()
	_, err := store.ping()
	if err == nil {
		t.Error("Expected error")
	}
}

func ExampleRediStore() {
	// RedisStore
	store, err := NewRediStore(10, "tcp", ":6379", "", []byte("secret-key"))
	if err != nil {
		panic(err)
	}
	defer store.Close()
}

func init() {
	gob.Register(FlashMessage{})
}
