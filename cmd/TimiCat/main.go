package main

import (
	"log"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/handler"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/model"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/repository"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/service"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 1️ 连接数据库
	dsn := "root:20070714@tcp(localhost:3306)/timi_cat?charset=utf8mb4&parseTime=True&loc=Local" //本地连接，要运行自行更改
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("无法连接数据库:", err)
	}

	// 自动迁移
	err = db.AutoMigrate(&model.User{})
	if err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	// 2️ 初始化依赖
	repo := repository.NewUserRepository(db)
	svc := service.NewUserService(repo)
	h := handler.NewUserHandler(svc)

	// 3️ 设置路由
	r := gin.Default()
	user := r.Group("/user")
	{
		user.POST("/guest_login", h.GuestLogin) //得到用户id，用户名，和token
	}

	// 使用Apifox测试时请将登录后得到的token放入请求头header中
	protected := r.Group("/focus")
	protected.Use(middleware.JWTAuth())
	{
		protected.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{"msg": "JWT验证成功", "user": c.GetString("user_name")})
		})
	}
	protected.POST("/start", h.StartFocus)
	protected.POST("/end", h.EndFocus)
	protected.GET("/ach", h.GetAchievement)

	// 4️ 启动服务
	r.Run(":8080")

}
