package handler

import (
	"net/http"
	"strconv"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/service"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {

	return &UserHandler{service: s}
}

// GuestLogin 游客登录
func (h *UserHandler) GuestLogin(c *gin.Context) {
	data, err := h.service.GuestLogin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "游客登录失败"})
		return
	}
	c.JSON(http.StatusOK, data)
}

// Create 创建用户 POST /focus/create
//func (h *UserHandler) Create(c *gin.Context) {
//	userID, _ := strconv.ParseUint(c.Query("id"), 10, 64)
//	_, err := h.service.Create(userID)
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//		return
//	}
//	c.JSON(http.StatusOK, gin.H{"message": "创建成功"})
//}

// StartFocus POST /focus/start
func (h *UserHandler) StartFocus(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("id"), 10, 64)
	record, err := h.service.StartFocus(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":   "开始专注",
		"id":        record.ID,
		"user_name": record.UserName,
	})
}

// EndFocus POST /focus/end
func (h *UserHandler) EndFocus(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("id"), 10, 64)
	record, err := h.service.EndFocus(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":         "专注结束",
		"user_name":       record.UserName,
		"duration_second": record.DurationSecond,
		"TotalDuration":   strconv.FormatInt(record.TotalDuration/60, 10) + "分钟",
	})
}

func (h *UserHandler) GetAchievement(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("id"), 10, 64)
	achs, err := h.service.GetAchievements(userID)
	record, _ := h.service.GetInformation(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"总专注时间":       strconv.FormatInt(record.TotalDuration/60, 10) + "分钟",
		"Achievement": achs,
	})
}
