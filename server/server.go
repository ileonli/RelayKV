package main

import (
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
)

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin") //请求头部
		if origin != "" {
			//接收客户端发送的origin （重要！）
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			//服务器支持的所有跨域请求的方法
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			//允许跨域设置可以返回其他子段，可以自定义字段
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session")
			// 允许浏览器（客户端）可以解析的头部 （重要）
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")
			//设置缓存时间
			c.Header("Access-Control-Max-Age", "172800")
			//允许客户端传递校验信息比如 cookie (重要)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		//允许类型校验
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, "ok!")
		}
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic info is: %v", err)
			}
		}()
		c.Next()
	}
}

func NewHttpService(httpAddress string, s *KVServer) {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard

	r := gin.Default()
	r.Use(cors())

	r.GET("status", func(c *gin.Context) {
		c.JSON(200, s.Status())
	})

	r.GET("GET", func(c *gin.Context) {
		key := c.Query("KEY")
		if key == "" {
			c.JSON(500, "")
			return
		}
		c.JSON(200, s.Get(key))
	})

	r.GET("PUT", func(c *gin.Context) {
		key := c.Query("KEY")
		value := c.Query("VALUE")

		if key == "" {
			c.JSON(500, "KEY can not be null")
			return
		}
		s.Put(key, value)
	})

	r.GET("DEL", func(c *gin.Context) {
		key := c.Query("KEY")

		if key == "" {
			c.JSON(500, "KEY can not be null")
			return
		}
		s.Delete(key)
	})

	if err := r.Run(httpAddress); err != nil {
		panic(err)
	}
}
