package database

import (
	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/config"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitGorm 初始化 GORM 数据库连接并运行自动迁移
// AutoMigrate 会自动创建表、添加缺失的列、创建约束和索引
// 若表已存在，只会添加新字段或修改字段（不会删除字段）
func InitGorm(cfg *config.Config) (*gorm.DB, error) {
	// 使用 PostgreSQL 驱动打开数据库连接
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	// 自动迁移这三个模型对应的表结构
	// Session：计时会话；Segment：计时片段；GrowthEvent：成长事件
	if err := db.AutoMigrate(&models.Session{}, &models.Segment{}, &models.GrowthEvent{}); err != nil {
		return nil, err
	}
	return db, nil
}
