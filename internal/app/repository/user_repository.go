package repository

import (
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		DB: db,
	}
}

// CreateUser  创建用户
func (r *UserRepository) CreateUser(user *model.User) error {
	return r.DB.Create(user).Error
}

func (r *UserRepository) GetUserByName(username string) (*model.User, error) {
	var user model.User
	err := r.DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, err
}

// GetInformation  （获取记录）
func (r *UserRepository) GetInformation(userID uint64) (*model.User, error) {
	var user model.User
	// 进行数据库查找时，传入结构体对象去赋值：.Find(&rec).Error
	err := r.DB.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, err
}

// Update 更新记录
func (r *UserRepository) Update(UserId uint64, user *model.User) error {
	return r.DB.Where("id = ? ", UserId).Updates(user).Error
}
