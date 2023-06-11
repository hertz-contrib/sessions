package main

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/sessions"
	"github.com/hertz-contrib/sessions/cookie"
	"sync"
)

var once sync.Once

func main() {
	h := server.New(server.WithHostPorts(":8000"))
	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{MaxAge: 10})
	h.Use(sessions.New("mysession", store))

	h.GET("/login", func(ctx context.Context, c *app.RequestContext) {
		s := sessions.Default(c)
		once.Do(func() {
			s.Set("key", "value")
			s.Options(sessions.Options{MaxAge: -1})
			s.Save()
		})
		i := s.Get("key")
		c.JSON(200, utils.H{
			"key": i,
		})
	}, sessions.Refresh(store))
	h.Spin()
}
