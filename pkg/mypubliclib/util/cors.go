package util

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// Cors CORS 中间件：允许配置的源（本地前端开发用） 从环境变量读取允许列表，只有在列表内的请求才会获得 CORS 头
func Cors() gin.HandlerFunc {
	allow := os.Getenv("ALLOW_ORIGINS")
	if allow == "" {
		// 默认允许常见本地开发地址（localhost/127.0.0.1 的 3000 与 5173 端口）
		allow = "http://localhost:3000,http://127.0.0.1:3000,http://localhost:5173,http://127.0.0.1:5173"
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		// 遍历允许列表，若请求来源在列表内则设置 CORS 头
		for _, a := range splitCSV(allow) {
			if origin == a {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
				c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				break
			}
		}
		// 对 OPTIONS 预检请求直接返回 204 No Content（浏览器跨域需要）
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

// splitCSV 将以逗号分隔的字符串分割成数组，并去除每个元素的前后空格
// 例如："http://localhost:3000, http://127.0.0.1:3000" 会被拆成两个元素
func splitCSV(s string) []string {
	out := []string{}
	t := ""
	for _, r := range s {
		if r == ',' {
			// 遇到逗号，把当前累积的字符串 trim 后加入结果
			if t != "" {
				out = append(out, trim(t))
			}
			t = ""
		} else {
			// 积累字符
			t += string(r)
		}
	}
	// 处理最后一个元素（最后面可能没有逗号）
	if t != "" {
		out = append(out, trim(t))
	}
	return out
}

// trim 从字符串的首尾删除空格和制表符
func trim(s string) string {
	// 删除前导空格和制表符
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	// 删除尾部空格和制表符
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
