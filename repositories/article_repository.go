package repositories

import (
	"cisdi-test-cms/models"
	"fmt"
	"log"

	"gorm.io/gorm"
)

type ArticleRepository interface {
	Create(article *models.Article) error
	GetByID(id uint) (*models.Article, error)
	GetList(params models.ArticleListParams, isPublic bool) ([]models.Article, int64, error)
	Update(article *models.Article) error
	Delete(id uint) error
	CreateVersion(version *models.ArticleVersion) error
	GetVersions(articleID uint) ([]models.ArticleVersion, error)
	GetVersion(articleID, versionID uint) (*models.ArticleVersion, error)
	UpdateVersion(id uint, updates map[string]interface{}) error
	GetVersionByID(versionID uint) (*models.ArticleVersion, error)
	CountTagPairs() (map[string]map[string]int, error)
	CountArticlesByTag() (map[uint]int, error)
	GetTagsForArticle(articleID int) ([]string, error)
	GetTotalArticleCount() (int64, error)
	GetArticleCountWithTag(tagName string) (int, error)
	GetArticleCountWithTags(tag1, tag2 string) (int, error)
	ClearPublishedVersionID(articleID uint) error
}

type articleRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return &articleRepository{db: db}
}

func (r *articleRepository) Create(article *models.Article) error {
	return r.db.Create(article).Error
}

func (r *articleRepository) GetByID(id uint) (*models.Article, error) {
	var article models.Article
	err := r.db.Preload("Author").
		Preload("PublishedVersion.Tags").
		Preload("LatestVersion.Tags").
		First(&article, id).Error
	return &article, err
}

func (r *articleRepository) GetList(params models.ArticleListParams, isPublic bool) ([]models.Article, int64, error) {
	var articles []models.Article
	var total int64

	query := r.db.Model(&models.Article{}).Preload("Author").Preload("LatestVersion.Tags")

	// Add public filter
	if isPublic {
		query = query.Joins("JOIN article_versions ON articles.published_version_id = article_versions.id").
			Where("article_versions.status = ?", models.StatusPublished)
	}

	// Add filters
	if params.Status != "" && !isPublic {
		query = query.Joins("JOIN article_versions ON articles.latest_version_id = article_versions.id").
			Where("article_versions.status = ?", params.Status)
	}

	if params.AuthorID > 0 {
		query = query.Where("author_id = ?", params.AuthorID)
	}

	if params.TagID > 0 {
		query = query.Joins("JOIN article_versions ON articles.latest_version_id = article_versions.id").
			Joins("JOIN article_version_tags ON article_versions.id = article_version_tags.article_version_id").
			Where("article_version_tags.tag_id = ?", params.TagID)
	}

	// Count total
	query.Count(&total)

	// Add sorting
	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}

	sortOrder := params.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	if sortBy == "article_tag_relationship_score" {
		query = query.Joins("JOIN article_versions ON articles.latest_version_id = article_versions.id").
			Order(fmt.Sprintf("article_versions.article_tag_relationship_score %s", sortOrder))
	} else {
		query = query.Order(fmt.Sprintf("articles.%s %s", sortBy, sortOrder))
	}

	// Add pagination
	offset := (params.Page - 1) * params.Limit
	err := query.Offset(offset).Limit(params.Limit).Find(&articles).Error

	return articles, total, err
}

func (r *articleRepository) Update(article *models.Article) error {
	return r.db.Save(article).Error
}

func (r *articleRepository) Delete(id uint) error {
	return r.db.Delete(&models.Article{}, id).Error
}

func (r *articleRepository) CreateVersion(version *models.ArticleVersion) error {
	return r.db.Create(version).Error
}

func (r *articleRepository) GetVersions(articleID uint) ([]models.ArticleVersion, error) {
	var versions []models.ArticleVersion
	err := r.db.Where("article_id = ?", articleID).
		Preload("Tags").
		Order("version_number desc").
		Find(&versions).Error
	return versions, err
}

func (r *articleRepository) GetVersion(articleID, versionID uint) (*models.ArticleVersion, error) {
	var version models.ArticleVersion
	err := r.db.Where("article_id = ? AND id = ?", articleID, versionID).
		Preload("Tags").
		First(&version).Error
	return &version, err
}

func (r *articleRepository) UpdateVersion(id uint, updates map[string]interface{}) error {
	return r.db.Model(&models.ArticleVersion{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *articleRepository) GetVersionByID(versionID uint) (*models.ArticleVersion, error) {
	var version models.ArticleVersion
	err := r.db.Preload("Tags").First(&version, versionID).Error
	return &version, err
}

func (r *articleRepository) CountTagPairs() (map[string]map[string]int, error) {
	var results []struct {
		Tag1Name string
		Tag2Name string
		Count    int
	}

	query := `
		SELECT 
			t1.name as tag1_name,
			t2.name as tag2_name,
			COUNT(*) as count
		FROM article_version_tags avt1
		JOIN article_version_tags avt2 ON avt1.article_version_id = avt2.article_version_id AND avt1.tag_id < avt2.tag_id
		JOIN tags t1 ON avt1.tag_id = t1.id
		JOIN tags t2 ON avt2.tag_id = t2.id
		JOIN article_versions av ON avt1.article_version_id = av.id
		WHERE av.status = 'published'
		GROUP BY t1.name, t2.name
	`

	err := r.db.Raw(query).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	tagPairs := make(map[string]map[string]int)
	for _, result := range results {
		if tagPairs[result.Tag1Name] == nil {
			tagPairs[result.Tag1Name] = make(map[string]int)
		}
		if tagPairs[result.Tag2Name] == nil {
			tagPairs[result.Tag2Name] = make(map[string]int)
		}
		tagPairs[result.Tag1Name][result.Tag2Name] = result.Count
		tagPairs[result.Tag2Name][result.Tag1Name] = result.Count
	}

	return tagPairs, nil
}

func (r *articleRepository) CountArticlesByTag() (map[uint]int, error) {
	var results []struct {
		TagID uint
		Count int
	}

	query := `
		SELECT 
			avt.tag_id,
			COUNT(*) as count
		FROM article_version_tags avt
		JOIN article_versions av ON avt.article_version_id = av.id
		WHERE av.status = 'published'
		GROUP BY avt.tag_id
	`

	err := r.db.Raw(query).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[uint]int)
	for _, result := range results {
		counts[result.TagID] = result.Count
	}

	return counts, nil
}

func (r *articleRepository) GetTagsForArticle(articleID int) ([]string, error) {
	var tags []string

	const query = `
		SELECT t.name
		FROM articles a
		JOIN article_versions av ON av.id = a.latest_version_id
		JOIN article_version_tags avt ON avt.article_version_id = av.id
		JOIN tags t ON t.id = avt.tag_id
		WHERE a.id = $1
		  AND a.deleted_at IS NULL
		  AND av.deleted_at IS NULL
		  AND t.deleted_at IS NULL
		ORDER BY t.name;
	`

	err := r.db.Raw(query, articleID).Scan(&tags).Error
	if err != nil {
		log.Printf("error fetching tags for article %d: %v", articleID, err)
		return nil, err
	}

	return tags, nil
}

func (r *articleRepository) GetTotalArticleCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.Article{}).Where("deleted_at IS NULL").Count(&count).Error
	if err != nil {
		log.Printf("error counting total articles: %v", err)
		return 0, err
	}
	return count, nil
}

func (r *articleRepository) GetArticleCountWithTag(tagName string) (int, error) {
	var count int

	const query = `
		SELECT COUNT(DISTINCT a.id)
		FROM articles a
		JOIN article_versions av ON av.id = a.latest_version_id
		JOIN article_version_tags avt ON avt.article_version_id = av.id
		JOIN tags t ON t.id = avt.tag_id
		WHERE t.name = $1
		  AND a.deleted_at IS NULL
		  AND av.deleted_at IS NULL
		  AND t.deleted_at IS NULL;
	`

	err := r.db.Raw(query, tagName).Scan(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *articleRepository) GetArticleCountWithTags(tag1, tag2 string) (int, error) {
	var count int

	const query = `
		SELECT COUNT(DISTINCT a.id)
		FROM articles a
		JOIN article_versions av ON av.id = a.latest_version_id
		JOIN article_version_tags avt1 ON avt1.article_version_id = av.id
		JOIN tags t1 ON t1.id = avt1.tag_id
		JOIN article_version_tags avt2 ON avt2.article_version_id = av.id
		JOIN tags t2 ON t2.id = avt2.tag_id
		WHERE a.deleted_at IS NULL
		  AND av.deleted_at IS NULL
		  AND t1.deleted_at IS NULL
		  AND t2.deleted_at IS NULL
		  AND t1.name = $1
		  AND t2.name = $2;
	`

	err := r.db.Raw(query, tag1, tag2).Scan(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *articleRepository) ClearPublishedVersionID(articleID uint) error {
	return r.db.Model(&models.Article{}).Where("id = ?", articleID).Update("published_version_id", nil).Error
}
