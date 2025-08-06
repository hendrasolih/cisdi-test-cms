package models

import (
	"time"

	"gorm.io/gorm"
)

type VersionStatus string

const (
	StatusDraft           VersionStatus = "draft"
	StatusPublished       VersionStatus = "published"
	StatusArchivedVersion VersionStatus = "archived_version"
)

type ArticleVersion struct {
	ID                          uint           `json:"id" gorm:"primarykey"`
	ArticleID                   uint           `json:"article_id" gorm:"not null"`
	Article                     *Article       `json:"article,omitempty" gorm:"foreignKey:ArticleID"`
	VersionNumber               int            `json:"version_number" gorm:"not null"`
	Title                       string         `json:"title" gorm:"not null"`
	Content                     string         `json:"content" gorm:"type:text"`
	Status                      VersionStatus  `json:"status" gorm:"default:'draft'"`
	ArticleTagRelationshipScore float64        `json:"article_tag_relationship_score" gorm:"default:0"`
	Tags                        []Tag          `json:"tags" gorm:"many2many:article_version_tags;"`
	PublishedAt                 *time.Time     `json:"published_at"`
	CreatedAt                   time.Time      `json:"created_at"`
	UpdatedAt                   time.Time      `json:"updated_at"`
	DeletedAt                   gorm.DeletedAt `json:"-" gorm:"index"`
}
