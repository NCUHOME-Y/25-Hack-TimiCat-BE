package service

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/model"
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/repository"
	utils "github.com/NCUHOME-Y/25-Hack-TimiCat-BE/pkg/mypubliclib/util"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// GuestLogin 生成游客账号
func (s *UserService) GuestLogin() (map[string]interface{}, error) {
	userName := fmt.Sprintf("guest_%d", time.Now().UnixMilli())
	user := &model.User{
		UserName:  userName,
		StartTime: nil,
		IsGuest:   true,
	}
	if err := s.repo.CreateUser(user); err != nil {
		return nil, err
	}

	token, err := utils.GenerateToken(user.ID, user.UserName, true)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":        user.ID,
		"user_name": userName,
		"token":     token,
	}, nil
}

// Create 创建记录
//func (s *UserService) Create(userID uint64) (*model.User, error) {
//	record := &model.User{
//		ID:        userID,
//		StartTime: nil,
//	}
//	if err := s.repo.Create(record); err != nil {
//		return nil, err
//	}
//	return record, nil
//}

// StartFocus 开始计时
func (s *UserService) StartFocus(userID uint64) (*model.User, error) {
	now := time.Now()

	record := &model.User{
		ID:             userID,
		StartTime:      &now,
		IsCompleted:    false,
		DurationSecond: 0,
	}
	if err := s.repo.Update(userID, record); err != nil {
		return nil, err
	}
	return record, nil
}

// EndFocus 结束计时
func (s *UserService) EndFocus(userID uint64) (*model.User, error) {
	rec, err := s.repo.GetInformation(userID)
	if rec != nil {
	} else {
		log.Printf("err: %v", err)
		return nil, err
	}
	if err != nil {
		log.Printf("err: %v", err)
		return nil, err
	}
	if rec.IsCompleted {
		return rec, errors.New("专注已结束")
	} else {
		now := time.Now()
		rec.EndTime = &now
		rec.DurationSecond = int64(now.Sub(*rec.StartTime).Seconds()) // 通过开始结束的时间相减来判断总时长（可能需要修改）
		rec.TotalDuration += rec.DurationSecond
		rec.IsCompleted = true
		return rec, s.repo.Update(userID, rec)
	}
}

// GetAchievements 获取所有成就
func (s *UserService) GetAchievements(userID uint64) ([]model.Achievement, error) {
	rec, err := s.repo.GetInformation(userID)
	if err != nil {
		log.Printf("err: %v", err)
		return nil, err
	}
	// 粗糙编码，后续可能需要更新
	if rec.TotalDuration > 1 && rec.TotalDuration < 5*3600+20*60 {
		rec.Achievement = "0001"
	} else if rec.TotalDuration > 5*3600+20*60 && rec.TotalDuration < 24*3600 {
		rec.Achievement = "0011"
	} else if rec.TotalDuration > 24*3600 && rec.TotalDuration < 36*3600 {
		rec.Achievement = "0111"
	} else if rec.TotalDuration > 36*3600 {
		rec.Achievement = "1111"
	}
	achBitMap, err := strconv.Atoi(rec.Achievement) // 位图记录成就
	if err != nil {
		log.Printf("成就错误: %v", err)
		return nil, err
	}

	achs := make([]model.Achievement, 0)
	tmp, idx := achBitMap, 0
	for tmp > 0 {
		if tmp&1 == 1 {
			achs = append(achs, model.Achievements[idx])
		}
		tmp = tmp >> 1
		idx++
	}
	return achs, nil
}

// 获取用户信息
func (s *UserService) GetInformation(userID uint64) (*model.User, error) {
	rec, err := s.repo.GetInformation(userID)
	if err != nil {
		log.Printf("err: %v", err)
		return nil, err
	}
	return rec, nil
}
