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

package tester

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/hertz-contrib/sessions"
)

type storeFactory func(*testing.T) sessions.Store

const sessionName = "mysession"

const ok = "ok"

func GetSet(t *testing.T, newStore storeFactory) {
	opt := config.NewOptions([]config.Option{})
	r := route.NewEngine(opt)
	r.Use(sessions.New(sessionName, newStore(t)))
	r.GET("/set", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		session.Set("key", ok)
		_ = session.Save()
		c.String(consts.StatusOK, ok)
	})

	r.GET("/get", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		if session.Get("key") != ok {
			t.Error("Session writing failed")
		}
		_ = session.Save()
		c.String(http.StatusOK, ok)
	})

	w1 := ut.PerformRequest(r, consts.MethodGet, "/set", nil)
	res1 := w1.Result()

	_ = ut.PerformRequest(r, consts.MethodGet, "/get", nil, ut.Header{
		Key:   "Cookie",
		Value: strings.Join(adaptor.GetCompatResponseWriter(res1).Header().Values("Set-Cookie"), "; "),
	})
}

func DeleteKey(t *testing.T, newStore storeFactory) {
	opt := config.NewOptions([]config.Option{})
	r := route.NewEngine(opt)
	r.Use(sessions.New(sessionName, newStore(t)))
	r.GET("/set", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		session.Set("key", ok)
		_ = session.Save()
		c.String(consts.StatusOK, ok)
	})
	r.GET("/delete", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		session.Delete("key")
		_ = session.Save()
		c.String(http.StatusOK, ok)
	})
	r.GET("/get", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		if session.Get("key") != nil {
			t.Error("Session deleting failed")
		}
		_ = session.Save()
		c.String(http.StatusOK, ok)
	})

	w1 := ut.PerformRequest(r, consts.MethodGet, "/set", nil)
	res1 := w1.Result()
	w2 := ut.PerformRequest(r, consts.MethodGet, "/delete", nil, ut.Header{
		Key:   "Cookie",
		Value: strings.Join(adaptor.GetCompatResponseWriter(res1).Header().Values("Set-Cookie"), "; "),
	})
	res2 := w2.Result()
	_ = ut.PerformRequest(r, consts.MethodGet, "/get", nil, ut.Header{
		Key:   "Cookie",
		Value: strings.Join(adaptor.GetCompatResponseWriter(res2).Header().Values("Set-Cookie"), "; "),
	})
}

func Flashes(t *testing.T, newStore storeFactory) {
	opt := config.NewOptions([]config.Option{})
	r := route.NewEngine(opt)
	r.Use(sessions.New(sessionName, newStore(t)))

	r.GET("/set", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		session.AddFlash(ok)
		_ = session.Save()
		c.String(http.StatusOK, ok)
	})

	r.GET("/flash", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		l := len(session.Flashes())
		if l != 1 {
			t.Error("Flashes count does not equal 1. Equals ", l)
		}
		_ = session.Save()
		c.String(http.StatusOK, ok)
	})

	r.GET("/check", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		l := len(session.Flashes())
		if l != 0 {
			t.Error("flashes count is not 0 after reading. Equals ", l)
		}
		_ = session.Save()
		c.String(http.StatusOK, ok)
	})

	w1 := ut.PerformRequest(r, consts.MethodGet, "/set", nil)
	res1 := w1.Result()

	w2 := ut.PerformRequest(r, consts.MethodGet, "/flash", nil, ut.Header{
		Key:   "Cookie",
		Value: strings.Join(adaptor.GetCompatResponseWriter(res1).Header().Values("Set-Cookie"), "; "),
	})
	res2 := w2.Result()

	_ = ut.PerformRequest(r, consts.MethodGet, "/check", nil, ut.Header{
		Key:   "Cookie",
		Value: strings.Join(adaptor.GetCompatResponseWriter(res2).Header().Values("Set-Cookie"), "; "),
	})
}

func Clear(t *testing.T, newStore storeFactory) {
	data := map[string]string{
		"key": "val",
		"foo": "bar",
	}
	opt := config.NewOptions([]config.Option{})
	r := route.NewEngine(opt)
	r.Use(sessions.New(sessionName, newStore(t)))

	r.GET("/set", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		for k, v := range data {
			session.Set(k, v)
		}
		session.Clear()
		_ = session.Save()
		c.String(http.StatusOK, ok)
	})

	r.GET("/check", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		for k, v := range data {
			if session.Get(k) == v {
				t.Fatal("Session clear failed")
			}
		}
		_ = session.Save()
		c.String(http.StatusOK, ok)
	})

	w1 := ut.PerformRequest(r, consts.MethodGet, "/set", nil)
	res1 := w1.Result()
	_ = ut.PerformRequest(r, consts.MethodGet, "/check", nil, ut.Header{
		Key:   "Cookie",
		Value: strings.Join(adaptor.GetCompatResponseWriter(res1).Header().Values("Set-Cookie"), "; "),
	})
}

func Options(t *testing.T, newStore storeFactory) {
	opt := config.NewOptions([]config.Option{})
	r := route.NewEngine(opt)
	store := newStore(t)
	store.Options(sessions.Options{
		Domain: "localhost",
	})
	r.Use(sessions.New(sessionName, store))
	r.GET("/domain", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		session.Set("key", ok)
		session.Options(sessions.Options{
			Path: "/foo/bar/bat",
		})
		_ = session.Save()
		c.String(http.StatusOK, ok)
	})
	r.GET("/path", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		session.Set("key", ok)
		_ = session.Save()
		c.String(http.StatusOK, ok)
	})
	r.GET("/set", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		session.Set("key", ok)
		_ = session.Save()
		c.String(http.StatusOK, ok)
	})
	r.GET("/expire", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		session.Options(sessions.Options{
			MaxAge: -1,
		})
		_ = session.Save()
		c.String(http.StatusOK, ok)
	})
	r.GET("/check", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		val := session.Get("key")
		if val != nil {
			t.Fatal("Session expiration failed")
		}
		c.String(http.StatusOK, ok)
	})
	testOptionSameSitego(t, r)

	w1 := ut.PerformRequest(r, consts.MethodGet, "/domain", nil)
	res1 := adaptor.GetCompatResponseWriter(w1.Result())
	w2 := ut.PerformRequest(r, consts.MethodGet, "/path", nil)
	res2 := adaptor.GetCompatResponseWriter(w2.Result())
	_ = ut.PerformRequest(r, consts.MethodGet, "/set", nil)
	_ = ut.PerformRequest(r, consts.MethodGet, "/expire", nil)
	_ = ut.PerformRequest(r, consts.MethodGet, "/check", nil)
	for _, c := range res1.Header().Values("Set-Cookie") {
		s := strings.Split(c, ";")
		if s[1] != " path=/foo/bar/bat" {
			t.Error("Error writing path with options:", s[1])
		}
	}

	for _, c := range res2.Header().Values("Set-Cookie") {
		s := strings.Split(c, ";")
		if s[1] != " domain=localhost" {
			t.Error("Error writing domain with options:", s[1])
		}
	}
}

func Many(t *testing.T, newStore storeFactory) {
	opt := config.NewOptions([]config.Option{})
	r := route.NewEngine(opt)
	sessionNames := []string{"a", "b"}
	r.Use(sessions.Many(sessionNames, newStore(t)))
	r.GET("/set", func(ctx context.Context, c *app.RequestContext) {
		sessionA := sessions.DefaultMany(c, "a")
		sessionA.Set("hello", "world")
		_ = sessionA.Save()

		sessionB := sessions.DefaultMany(c, "b")
		sessionB.Set("foo", "bar")
		_ = sessionB.Save()
		c.String(http.StatusOK, ok)
	})

	r.GET("/get", func(ctx context.Context, c *app.RequestContext) {
		sessionA := sessions.DefaultMany(c, "a")
		if sessionA.Get("hello") != "world" {
			t.Error("Session writing failed")
		}
		_ = sessionA.Save()

		sessionB := sessions.DefaultMany(c, "b")
		if sessionB.Get("foo") != "bar" {
			t.Error("Session writing failed")
		}
		_ = sessionB.Save()
		c.String(http.StatusOK, ok)
	})

	w1 := ut.PerformRequest(r, consts.MethodGet, "/set", nil)
	res1 := adaptor.GetCompatResponseWriter(w1.Result())
	header := ""
	for _, x := range res1.Header()["Set-Cookie"] {
		header += strings.Split(x, ";")[0] + "; \n"
	}
	_ = ut.PerformRequest(r, consts.MethodGet, "/get", nil, ut.Header{
		Key:   "Cookie",
		Value: header,
	})
}
