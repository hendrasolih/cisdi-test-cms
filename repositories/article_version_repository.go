package repositories

import (
	"cisdi-test-cms/models"

	"gorm.io/gorm"
)

type ArticleVersionRepository interface {
	DeleteVersionsByArticleID(articleID uint) error
}

type articleVersionRepository struct {
	db *gorm.DB
}

func NewArticleVersionRepository(db *gorm.DB) ArticleVersionRepository {
	return &articleVersionRepository{db: db}
}

func (r *articleVersionRepository) DeleteVersionsByArticleID(articleID uint) error {
	return r.db.Where("article_id = ?", articleID).Delete(&models.ArticleVersion{}).Error
}
