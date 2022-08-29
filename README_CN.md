# sessions (这是一个社区驱动的项目)
[English](README.md) | 中文

这是 Hertz 的一个中间件。

是一个用于管理 session 的 Hertz 中间件，支持 multi-backend。 

- [cookie-based](#cookie-based)
- [Redis](#redis)

## 使用方法

### 开始使用

如何下载并安装它:

```bash
go get github.com/hertz-contrib/sessions
```

如何导入进你的代码:

```go
import "github.com/hertz-contrib/sessions"
```

## 基本示例

### 单个 session
```go
package main

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/sessions"
	"github.com/hertz-contrib/sessions/cookie"
)

func main() {
	h := server.New(server.WithHostPorts(":8000"))
	store := cookie.NewStore([]byte("secret"))
	h.Use(sessions.Sessions("mysession", store))
	h.GET("/hello", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		
		if session.Get("hello") != "world" {
			session.Set("hello", "world")
			session.Save()
		}
		
		c.JSON(200, utils.H{"hello": session.Get("hello")})
	})
	h.Spin()
}
```

### 多个 session
```go
package main

import (
	"context"
	
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/sessions"
	"github.com/hertz-contrib/sessions/cookie"
)

func main() {
	h := server.New(server.WithHostPorts(":8000"))
	store := cookie.NewStore([]byte("secret"))
	sessionNames := []string{"a", "b"}
	h.Use(sessions.Sessions(sessionNames, store))
	h.GET("/hello", func(ctx context.Context, c *app.RequestContext) {
		sessionA := sessions.DefaultMany(c, "a")
		sessionB := sessions.DefaultMany(c, "b")
		
		if sessionA.Get("hello") != "world!" {
			sessionA.Set("hello", "world!")
			sessionA.Save()
		}
		
		if sessionB.Get("hello") != "world?" {
			sessionB.Set("hello", "world?")
			sessionB.Save()
		}
		
		c.JSON(200, utils.H{
			"a": sessionA.Get("hello"),
			"b": sessionB.Get("hello"),
		})
	})
	h.Spin()
}
```
## 后台实例

### cookie-based

```go
package main

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/sessions"
	"github.com/hertz-contrib/sessions/cookie"
)

func main() {
	h := server.New(server.WithHostPorts(":8000"))
	store := cookie.NewStore([]byte("secret"))
	h.Use(sessions.Sessions("mysession", store))
	h.GET("/incr", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		var count int
		v := session.Get("count")
		if v == nil {
			count = 0
		} else {
			count = v.(int)
			count++
		}
		session.Set("count", count)
		session.Save()
		c.JSON(200, utils.H{"count": count})
	})
	h.Spin()
}
```

### Redis

```go
package main

import (
	"context"
	
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/sessions"
	"github.com/hertz-contrib/sessions/redis"
)

func main() {
	h := server.Default(server.WithHostPorts(":8000"))
	store, _ := redis.NewStore(10, "tcp", "localhost:6379", "", []byte("secret"))
	h.Use(sessions.Sessions("mysession", store))
	
	h.GET("/incr", func(ctx context.Context, c *app.RequestContext) {
		session := sessions.Default(c)
		var count int
		v := session.Get("count")
		if v == nil {
			count = 0
		} else {
			count = v.(int)
			count++
		}
		session.Set("count", count)
		session.Save()
		c.JSON(200, utils.H{"count": count})
	})
	h.Spin()
}
```

## 许可证

本项目采用Apache许可证。参见 [LICENSE](LICENSE) 文件中的完整许可证文本。
