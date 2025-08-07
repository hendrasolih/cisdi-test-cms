package services

import (
	"errors"
	"fmt"
	"math"
	"time"

	"cisdi-test-cms/models"
	"cisdi-test-cms/repositories"

	"gorm.io/gorm"
)

type ArticleService interface {
	CreateArticle(req models.CreateArticleRequest, userID uint) (*models.Article, error)
	GetArticle(id uint, userID uint, isPublic bool) (*models.Article, error)
	GetArticles(params models.ArticleListParams, userID uint, isPublic bool) ([]models.Article, int64, error)
	DeleteArticle(id uint, userID uint) error
	CreateArticleVersion(articleID uint, req models.CreateArticleVersionRequest, userID uint) (*models.ArticleVersion, error)
	UpdateVersionStatus(articleID, versionID uint, status models.VersionStatus, userID uint) error
	GetArticleVersions(articleID uint, userID uint) ([]models.ArticleVersion, error)
	GetArticleVersion(articleID, versionID uint, userID uint) (*models.ArticleVersion, error)
}

type articleService struct {
	articleRepo repositories.ArticleRepository
	tagRepo     repositories.TagRepository
}

func NewArticleService(articleRepo repositories.ArticleRepository, tagRepo repositories.TagRepository) ArticleService {
	return &articleService{
		articleRepo: articleRepo,
		tagRepo:     tagRepo,
	}
}

func (s *articleService) CreateArticle(req models.CreateArticleRequest, userID uint) (*models.Article, error) {
	// Process tags
	tags, err := s.processTagsForVersion(req.Tags)
	if err != nil {
		return nil, err
	}

	// Create article
	article := &models.Article{
		AuthorID: userID,
		Title:    req.Title,
	}

	// Create first version
	version := &models.ArticleVersion{
		VersionNumber: 1,
		Title:         req.Title,
		Content:       req.Content,
		Status:        models.StatusDraft,
		Tags:          tags,
	}

	// Calculate article tag relationship score
	version.ArticleTagRelationshipScore = s.calculateArticleTagRelationshipScoreCreateArticle(req.Tags)

	// Set article version relationships
	article.LatestVersionID = 0 // Will be updated after creation

	// Create article first, then version
	if err := s.articleRepo.Create(article); err != nil {
		return nil, err
	}

	version.ArticleID = article.ID
	if err := s.articleRepo.CreateVersion(version); err != nil {
		return nil, err
	}

	// Update article with version ID
	article.LatestVersionID = version.ID
	if err := s.articleRepo.Update(article); err != nil {
		return nil, err
	}

	// Update tag usage counts
	s.updateTagUsageCounts()

	// Load the complete article
	return s.articleRepo.GetByID(article.ID)
}

func (s *articleService) GetArticle(id uint, userID uint, isPublic bool) (*models.Article, error) {
	article, err := s.articleRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Check access permissions
	if isPublic && (article.PublishedVersion == nil || article.PublishedVersion.Status != models.StatusPublished) {
		return nil, errors.New("article not found")
	}

	if !isPublic && article.AuthorID != userID {
		// Allow editors and admins to view any article
		// This would need role checking in the handler
	}

	return article, nil
}

func (s *articleService) GetArticles(params models.ArticleListParams, userID uint, isPublic bool) ([]models.Article, int64, error) {
	return s.articleRepo.GetList(params, isPublic)
}

func (s *articleService) DeleteArticle(id uint, userID uint) error {
	article, err := s.articleRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Check ownership or admin/editor role (would need role in context)
	if article.AuthorID != userID {
		return errors.New("unauthorized")
	}

	return s.articleRepo.Delete(id)
}

func (s *articleService) CreateArticleVersion(articleID uint, req models.CreateArticleVersionRequest, userID uint) (*models.ArticleVersion, error) {
	// Check if article exists and user has access
	article, err := s.articleRepo.GetByID(articleID)
	if err != nil {
		return nil, err
	}

	if article.AuthorID != userID {
		return nil, errors.New("unauthorized")
	}

	// Get existing versions to determine next version number
	versions, err := s.articleRepo.GetVersions(articleID)
	if err != nil {
		return nil, err
	}

	nextVersionNumber := 1
	if len(versions) > 0 {
		nextVersionNumber = versions[0].VersionNumber + 1
	}

	// Process tags
	tags, err := s.processTagsForVersion(req.Tags)
	if err != nil {
		return nil, err
	}

	// Create new version
	version := &models.ArticleVersion{
		ArticleID:     articleID,
		VersionNumber: nextVersionNumber,
		Title:         req.Title,
		Content:       req.Content,
		Status:        models.StatusDraft,
		Tags:          tags,
	}

	articleIDint := int(articleID)

	// Calculate article tag relationship score
	version.ArticleTagRelationshipScore = s.calculateArticleTagRelationshipScoreCreateArticleVersion(articleIDint)

	if err := s.articleRepo.CreateVersion(version); err != nil {
		return nil, err
	}

	// Update article's latest version
	article.LatestVersionID = version.ID
	article.Title = req.Title // Update article title to match latest version
	if err := s.articleRepo.Update(article); err != nil {
		return nil, err
	}

	// Update tag usage counts
	s.updateTagUsageCounts()

	return s.articleRepo.GetVersionByID(version.ID)
}

func (s *articleService) UpdateVersionStatus(articleID, versionID uint, status models.VersionStatus, userID uint) error {
	// Check article access
	article, err := s.articleRepo.GetByID(articleID)
	if err != nil {
		return err
	}

	if article.AuthorID != userID {
		return errors.New("unauthorized")
	}

	// Get the version
	version, err := s.articleRepo.GetVersion(articleID, versionID)
	if err != nil {
		return err
	}

	// Handle status changes
	if status == models.StatusPublished {
		// If publishing this version, unpublish any currently published version
		if article.PublishedVersionID != nil && *article.PublishedVersionID != version.ID {
			currentPublished, err := s.articleRepo.GetVersionByID(*article.PublishedVersionID)
			if err == nil {
				currentPublished.Status = models.StatusArchivedVersion
				s.articleRepo.UpdateVersion(currentPublished)
			}
		}

		version.Status = status
		now := time.Now()
		version.PublishedAt = &now

		// Update article's published version
		article.PublishedVersionID = &version.ID
		if err := s.articleRepo.Update(article); err != nil {
			return err
		}
	} else {
		version.Status = status
		if status == models.StatusArchivedVersion && article.PublishedVersionID != nil && *article.PublishedVersionID == version.ID {
			article.PublishedVersionID = nil
			if err := s.articleRepo.Update(article); err != nil {
				return err
			}
		}
	}

	// Update tag usage counts after status change
	s.updateTagUsageCounts()

	return s.articleRepo.UpdateVersion(version)
}

func (s *articleService) GetArticleVersions(articleID uint, userID uint) ([]models.ArticleVersion, error) {
	// Check access
	article, err := s.articleRepo.GetByID(articleID)
	if err != nil {
		return nil, err
	}

	if article.AuthorID != userID {
		return nil, errors.New("unauthorized")
	}

	return s.articleRepo.GetVersions(articleID)
}

func (s *articleService) GetArticleVersion(articleID, versionID uint, userID uint) (*models.ArticleVersion, error) {
	// Check access
	article, err := s.articleRepo.GetByID(articleID)
	if err != nil {
		return nil, err
	}

	if article.AuthorID != userID {
		return nil, errors.New("unauthorized")
	}

	return s.articleRepo.GetVersion(articleID, versionID)
}

func (s *articleService) processTagsForVersion(tagNames []string) ([]models.Tag, error) {
	var tags []models.Tag

	for _, name := range tagNames {
		tag, err := s.tagRepo.GetByName(name)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Create new tag
				newTag := &models.Tag{
					Name:          name,
					UsageCount:    0,
					TrendingScore: 0,
				}
				if err := s.tagRepo.Create(newTag); err != nil {
					return nil, err
				}
				tags = append(tags, *newTag)
			} else {
				return nil, err
			}
		} else {
			tags = append(tags, *tag)
		}
	}

	return tags, nil
}

func (s *articleService) calculateArticleTagRelationshipScoreCreateArticle(tags []string) float64 {
	if len(tags) < 2 {
		return 0.0
	}
	for i, t := range tags {
		fmt.Printf("Tag %d: %s\n", i+1, t)
	}

	countArticles, err := s.articleRepo.GetTotalArticleCount()
	if err != nil {
		fmt.Println("Error getting total article count:", err)
		return 0.0
	}

	totalArticles := float64(countArticles)
	scoreSum := 0.0
	pairCount := 0

	for i := 0; i < len(tags)-1; i++ {
		for j := i + 1; j < len(tags); j++ {
			tag1 := tags[i]
			tag2 := tags[j]
			countTag1Int, err := s.articleRepo.GetArticleCountWithTag(tag1)
			if err != nil {
				fmt.Println("Error getting count for tag:", tag1, err)
				continue
			}
			countTag2Int, err := s.articleRepo.GetArticleCountWithTag(tag2)
			if err != nil {
				fmt.Println("Error getting count for tag:", tag2, err)
				continue
			}
			countBothInt, err := s.articleRepo.GetArticleCountWithTags(tag1, tag2)
			if err != nil {
				fmt.Println("Error getting count for tag pair:", tag1, tag2, err)
				continue
			}

			countTag1 := float64(countTag1Int)
			countTag2 := float64(countTag2Int)
			countBoth := float64(countBothInt)

			if countTag1 == 0 || countTag2 == 0 || countBoth == 0 {
				continue
			}

			pTag1 := countTag1 / totalArticles
			pTag2 := countTag2 / totalArticles
			pBoth := countBoth / totalArticles

			pmi := math.Log(pBoth / (pTag1 * pTag2))
			scoreSum += pmi
			pairCount++
		}
	}

	if pairCount == 0 {
		return 0.0
	}

	averageScore := scoreSum / float64(pairCount)
	return averageScore
}

// Fungsi utama: hitung skor hubungan antar tag
func (s *articleService) calculateArticleTagRelationshipScoreCreateArticleVersion(articleID int) float64 {
	tags, err := s.articleRepo.GetTagsForArticle(articleID)
	if err != nil {
		fmt.Println("Error getting tags for article:", err)
		return 0.0
	}
	if len(tags) < 2 {
		return 0.0
	}
	for i, t := range tags {
		fmt.Printf("Tag %d: %s\n", i+1, t)
	}

	countArticles, err := s.articleRepo.GetTotalArticleCount()
	if err != nil {
		fmt.Println("Error getting total article count:", err)
		return 0.0
	}

	totalArticles := float64(countArticles)
	scoreSum := 0.0
	pairCount := 0

	for i := 0; i < len(tags)-1; i++ {
		for j := i + 1; j < len(tags); j++ {
			tag1 := tags[i]
			tag2 := tags[j]
			countTag1Int, err := s.articleRepo.GetArticleCountWithTag(tag1)
			if err != nil {
				fmt.Println("Error getting count for tag:", tag1, err)
				continue
			}
			countTag2Int, err := s.articleRepo.GetArticleCountWithTag(tag2)
			if err != nil {
				fmt.Println("Error getting count for tag:", tag2, err)
				continue
			}
			countBothInt, err := s.articleRepo.GetArticleCountWithTags(tag1, tag2)
			if err != nil {
				fmt.Println("Error getting count for tag pair:", tag1, tag2, err)
				continue
			}

			countTag1 := float64(countTag1Int)
			countTag2 := float64(countTag2Int)
			countBoth := float64(countBothInt)

			if countTag1 == 0 || countTag2 == 0 || countBoth == 0 {
				continue
			}

			pTag1 := countTag1 / totalArticles
			pTag2 := countTag2 / totalArticles
			pBoth := countBoth / totalArticles

			pmi := math.Log(pBoth / (pTag1 * pTag2))
			scoreSum += pmi
			pairCount++
		}
	}

	if pairCount == 0 {
		return 0.0
	}

	averageScore := scoreSum / float64(pairCount)
	return averageScore
}

func (s *articleService) updateTagUsageCounts() {
	// This would be called after any article version status change
	// Get all tags and their usage counts from published articles
	tagCounts, err := s.articleRepo.CountArticlesByTag()
	if err != nil {
		return
	}

	// Update all tags
	allTags, err := s.tagRepo.GetAll()
	if err != nil {
		return
	}

	for i := range allTags {
		if count, exists := tagCounts[allTags[i].ID]; exists {
			allTags[i].UsageCount = count
		} else {
			allTags[i].UsageCount = 0
		}

		// Calculate trending score (simple implementation)
		// This considers both usage count and recency
		daysSinceCreated := time.Since(allTags[i].CreatedAt).Hours() / 24
		if daysSinceCreated > 0 {
			allTags[i].TrendingScore = float64(allTags[i].UsageCount) / math.Log(daysSinceCreated+1)
		} else {
			allTags[i].TrendingScore = float64(allTags[i].UsageCount)
		}
	}

	s.tagRepo.BulkUpdate(allTags)
}
