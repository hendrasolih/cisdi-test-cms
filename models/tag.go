package models

import (
	"time"

	"gorm.io/gorm"
)

type Tag struct {
	ID            uint           `json:"id" gorm:"primarykey"`
	Name          string         `json:"name" gorm:"uniqueIndex;not null"`
	UsageCount    int            `json:"usage_count" gorm:"default:0"`
	TrendingScore float64        `json:"trending_score" gorm:"default:0"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}
