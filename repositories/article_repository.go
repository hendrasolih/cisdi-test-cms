package repositories

import (
	"cisdi-test-cms/models"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

type ArticleRepository interface {
	Create(article *models.Article) (*models.Article, error)
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
	UpdateFields(id uint, fields map[string]interface{}) error
	GetTagFrequencies(tagNames []string) (map[string]int, error)
	GetTagPairCoOccurrences(tagNames []string) (map[string]int, error)
}

type articleRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return &articleRepository{db: db}
}

func (r *articleRepository) Create(article *models.Article) (*models.Article, error) {
	if err := r.db.Create(article).Error; err != nil {
		return nil, err
	}
	return article, nil
}

func (r *articleRepository) GetByID(id uint) (*models.Article, error) {
	var article models.Article
	err := r.db.Preload("Author").
		Preload("PublishedVersion.Tags").
		Preload("LatestVersion.Tags").
		First(&article, id).Error
	return &article, err
}

// GetList mengambil daftar artikel dengan filter dan pagination sesuai params.
// Fungsi ini meng-handle dua mode utama:
// 1. Public mode (isPublic == true):
//    - Mengambil artikel yang sudah dipublikasikan,
//      yaitu artikel yang memiliki published_version_id dengan status "published".
//    - Menggunakan join ke tabel article_versions dengan alias av_pub pada published_version_id.
//    - Mengabaikan status versi terbaru (latest_version_id) yang bisa jadi masih draft.
// 2. Non-public mode (isPublic == false):
//    - Jika params.Status adalah "published", cari artikel berdasarkan published_version_id dan status published.
//    - Jika params.Status selain "published", cari artikel berdasarkan latest_version_id dengan status yang diberikan.
//    - Jika tidak ada status filter, join ke latest_version_id hanya jika perlu (misal sorting berdasarkan skor atau filter tag).
//
// Selain itu, fungsi ini juga menangani:
// - Filter berdasarkan AuthorID dan TagID, dengan join ke tabel tag yang sesuai alias article_versions yang aktif (av_pub atau av_lat).
// - Sorting berdasarkan field yang diminta, termasuk field khusus seperti article_tag_relationship_score.
// - Pagination dengan limit dan offset.
// - Debug print query SQL sebelum dijalankan untuk membantu proses debugging.
func (r *articleRepository) GetList(params models.ArticleListParams, isPublic bool) ([]models.Article, int64, error) {
	var articles []models.Article
	var total int64

	query := r.db.Model(&models.Article{}).
		Preload("Author").
		Preload("LatestVersion.Tags")

	if isPublic {
		// Public mode: hanya tampilkan artikel yang sudah published (published_version_id)
		query = query.Joins("JOIN article_versions av_pub ON articles.published_version_id = av_pub.id").
			Where("av_pub.status = ?", models.StatusPublished)
	} else {
		if params.Status == string(models.StatusPublished) {
			// Kalau status published, join ke published_version_id
			query = query.Joins("JOIN article_versions av_pub ON articles.published_version_id = av_pub.id").
				Where("av_pub.status = ?", models.StatusPublished)
		} else if params.Status != "" {
			// Kalau status selain published, join ke latest_version_id
			query = query.Joins("JOIN article_versions av_lat ON articles.latest_version_id = av_lat.id").
				Where("av_lat.status = ?", params.Status)
		} else {
			// Kalau tidak ada status filter, join latest_version_id jika perlu sorting atau filter tag
			if params.SortBy == "article_tag_relationship_score" || params.TagID > 0 {
				query = query.Joins("JOIN article_versions av_lat ON articles.latest_version_id = av_lat.id")
			}
		}
	}

	if params.AuthorID > 0 {
		query = query.Where("author_id = ?", params.AuthorID)
	}

	if params.TagID > 0 {
		// Pakai alias sesuai join yang aktif
		if params.Status == string(models.StatusPublished) || isPublic {
			query = query.Joins("JOIN article_version_tags avt ON av_pub.id = avt.article_version_id").
				Where("avt.tag_id = ?", params.TagID)
		} else {
			query = query.Joins("JOIN article_version_tags avt ON av_lat.id = avt.article_version_id").
				Where("avt.tag_id = ?", params.TagID)
		}
	}

	query.Count(&total)

	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}

	sortOrder := params.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	if sortBy == "article_tag_relationship_score" {
		if params.Status == string(models.StatusPublished) || isPublic {
			query = query.Order(fmt.Sprintf("av_pub.article_tag_relationship_score %s", sortOrder))
		} else {
			query = query.Order(fmt.Sprintf("av_lat.article_tag_relationship_score %s", sortOrder))
		}
	} else {
		query = query.Order(fmt.Sprintf("articles.%s %s", sortBy, sortOrder))
	}

	offset := (params.Page - 1) * params.Limit

	// Debug SQL
	stmt := query.Session(&gorm.Session{DryRun: true}).Offset(offset).Limit(params.Limit).Find(&articles).Statement
	fmt.Println("SQL:", stmt.SQL.String())
	fmt.Println("Vars:", stmt.Vars)

	err := query.Debug().Offset(offset).Limit(params.Limit).Find(&articles).Error

	return articles, total, err
}

func (r *articleRepository) Update(article *models.Article) error {
	return r.db.Save(article).Error
}

func (r *articleRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&models.Article{}).
		Where("id = ?", id).
		Updates(fields).
		Error
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
	fmt.Println("Updating version with ID:", id, "with updates:", updates)
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

type TagCheckRow struct {
	ArticleID      int
	VersionID      int
	TagID          int
	TagName        string
	ArticleDeleted *time.Time
	VersionDeleted *time.Time
	TagDeleted     *time.Time
}

func (r *articleRepository) GetTagFrequencies(tagNames []string) (map[string]int, error) {
	result := make(map[string]int)

	// Kalau tagNames kosong langsung return
	if len(tagNames) == 0 {
		return result, nil
	}

	// Buat placeholder ?,?,? sesuai jumlah tagNames
	placeholders := strings.Repeat("?,", len(tagNames))
	placeholders = strings.TrimRight(placeholders, ",")

	// Konversi []string ke []interface{} agar bisa di-spread di Raw()
	args := make([]interface{}, len(tagNames))
	for i, v := range tagNames {
		args[i] = v
	}

	// Susun query final
	query := fmt.Sprintf(`
		SELECT t.name, COUNT(DISTINCT a.id) AS freq
		FROM articles a
		JOIN article_versions av ON av.id = a.latest_version_id
		JOIN article_version_tags avt ON avt.article_version_id = av.id
		JOIN tags t ON t.id = avt.tag_id
		WHERE t.name IN (%s)
		  AND a.deleted_at IS NULL
		  AND av.deleted_at IS NULL
		  AND t.deleted_at IS NULL
		GROUP BY t.name
	`, placeholders)

	// Logging untuk debug
	log.Printf("[DEBUG] GetTagFrequencies query:\n%s\nArgs: %#v\n", query, args)

	// Jalankan query
	rows, err := r.db.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan hasil ke map
	for rows.Next() {
		var name string
		var freq int
		if err := rows.Scan(&name, &freq); err != nil {
			return nil, err
		}
		result[name] = freq
	}

	// Logging hasil untuk debug
	log.Printf("[DEBUG] GetTagFrequencies result: %#v\n", result)

	return result, nil
}

// GetTagPairCoOccurrences - ambil co-occurrence semua pasangan dalam 1 query
func (r *articleRepository) GetTagPairCoOccurrences(tagNames []string) (map[string]int, error) {
	result := make(map[string]int)
	if len(tagNames) < 2 {
		return result, nil
	}

	query := `
		SELECT LEAST(t1.name, t2.name) AS tag1,
		       GREATEST(t1.name, t2.name) AS tag2,
		       COUNT(DISTINCT a.id) AS freq
		FROM articles a
		JOIN article_versions av ON av.id = a.latest_version_id
		JOIN article_version_tags avt1 ON avt1.article_version_id = av.id
		JOIN tags t1 ON t1.id = avt1.tag_id
		JOIN article_version_tags avt2 ON avt2.article_version_id = av.id
		JOIN tags t2 ON t2.id = avt2.tag_id
		WHERE t1.name IN (?) 
		  AND t2.name IN (?) 
		  AND t1.name <> t2.name
		  AND a.deleted_at IS NULL
		  AND av.deleted_at IS NULL
		  AND t1.deleted_at IS NULL
		  AND t2.deleted_at IS NULL
		GROUP BY tag1, tag2
	`
	rows, err := r.db.Raw(query, tagNames, tagNames).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tag1, tag2 string
		var freq int
		if err := rows.Scan(&tag1, &tag2, &freq); err != nil {
			return nil, err
		}
		result[tag1+"|"+tag2] = freq
	}

	return result, nil
}
