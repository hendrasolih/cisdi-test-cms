package models

import "time"

type ArticleVersionTag struct {
	ID               uint      `json:"id" gorm:"primarykey"`
	ArticleVersionID uint      `json:"article_version_id"`
	TagID            uint      `json:"tag_id"`
	CreatedAt        time.Time `json:"created_at"`
}
