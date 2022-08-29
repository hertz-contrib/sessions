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

//go:build go1.11
// +build go1.11

package tester

import (
	ccontext "context"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/hertz-contrib/sessions"
)

func testOptionSameSitego(t *testing.T, r *route.Engine) {
	r.GET("/sameSite", func(ctx ccontext.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		session.Set("key", ok)
		session.Options(sessions.Options{
			SameSite: http.SameSiteStrictMode,
		})
		_ = session.Save()
		c.String(200, ok)
	})

	w1 := ut.PerformRequest(r, consts.MethodGet, "/sameSite", nil)
	res3 := w1.Result()
	resp3 := adaptor.GetCompatResponseWriter(res3)
	// res3 := httptest.NewRecorder()
	// req3, _ := http.NewRequest("GET", "/sameSite", nil)
	// r.ServeHTTP(res3, req3)

	s := strings.Split(resp3.Header().Get("Set-Cookie"), ";")
	if s[1] != " SameSite=Strict" {
		t.Error("Error writing same site with options:", s[1])
	}
}
