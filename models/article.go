package models

import (
	"time"

	"gorm.io/gorm"
)

type Article struct {
	ID                 uint             `json:"id" gorm:"primarykey"`
	AuthorID           uint             `json:"author_id" gorm:"not null"`
	Author             User             `json:"author" gorm:"foreignKey:AuthorID"`
	Title              string           `json:"title" gorm:"not null"`
	PublishedVersionID *uint            `json:"published_version_id"`
	PublishedVersion   *ArticleVersion  `json:"published_version,omitempty" gorm:"foreignKey:PublishedVersionID"`
	LatestVersionID    uint             `json:"latest_version_id"`
	LatestVersion      ArticleVersion   `json:"latest_version" gorm:"foreignKey:LatestVersionID"`
	Versions           []ArticleVersion `json:"versions,omitempty" gorm:"foreignKey:ArticleID"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	DeletedAt          gorm.DeletedAt   `json:"-" gorm:"index"`
}
