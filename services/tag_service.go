// services/tag_service.go
package services

import (
	"cisdi-test-cms/models"
	"cisdi-test-cms/repositories"
	"errors"

	"gorm.io/gorm"
)

type TagService interface {
	CreateTag(req models.CreateTagRequest) (*models.Tag, error)
	GetTags() ([]models.Tag, error)
	GetTag(id uint) (*models.Tag, error)
}

type tagService struct {
	tagRepo     repositories.TagRepository
	articleRepo repositories.ArticleRepository
}

func NewTagService(tagRepo repositories.TagRepository, articleRepo repositories.ArticleRepository) TagService {
	return &tagService{
		tagRepo:     tagRepo,
		articleRepo: articleRepo,
	}
}

func (s *tagService) CreateTag(req models.CreateTagRequest) (*models.Tag, error) {
	// Check if tag already exists
	_, err := s.tagRepo.GetByName(req.Name)
	if err == nil {
		return nil, errors.New("tag already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Create new tag
	tag := &models.Tag{
		Name:          req.Name,
		UsageCount:    0,
		TrendingScore: 0,
	}

	if err := s.tagRepo.Create(tag); err != nil {
		return nil, err
	}

	return tag, nil
}

func (s *tagService) GetTags() ([]models.Tag, error) {
	return s.tagRepo.GetAll()
}

func (s *tagService) GetTag(id uint) (*models.Tag, error) {
	return s.tagRepo.GetByID(id)
}
