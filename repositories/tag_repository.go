package repositories

import (
	"cisdi-test-cms/models"

	"gorm.io/gorm"
)

type TagRepository interface {
	Create(tag *models.Tag) error
	GetByName(name string) (*models.Tag, error)
	GetByNames(names []string) ([]models.Tag, error)
	GetByID(id uint) (*models.Tag, error)
	GetAll() ([]models.Tag, error)
	Update(tag *models.Tag) error
	BulkUpdate(tags []models.Tag) error
}

type tagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

func (r *tagRepository) Create(tag *models.Tag) error {
	return r.db.Create(tag).Error
}

func (r *tagRepository) GetByName(name string) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Where("name = ?", name).First(&tag).Error
	return &tag, err
}

func (r *tagRepository) GetByNames(names []string) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Where("name IN ?", names).Find(&tags).Error
	return tags, err
}

func (r *tagRepository) GetByID(id uint) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.First(&tag, id).Error
	return &tag, err
}

func (r *tagRepository) GetAll() ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Order("trending_score desc").Find(&tags).Error
	return tags, err
}

func (r *tagRepository) Update(tag *models.Tag) error {
	return r.db.Save(tag).Error
}

func (r *tagRepository) BulkUpdate(tags []models.Tag) error {
	return r.db.Save(&tags).Error
}
