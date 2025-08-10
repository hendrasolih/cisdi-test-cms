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
	articleRepo        repositories.ArticleRepository
	tagRepo            repositories.TagRepository
	articleVersionRepo repositories.ArticleVersionRepository
}

func NewArticleService(articleRepo repositories.ArticleRepository, tagRepo repositories.TagRepository, articleVersionRepo repositories.ArticleVersionRepository) ArticleService {
	return &articleService{
		articleRepo:        articleRepo,
		tagRepo:            tagRepo,
		articleVersionRepo: articleVersionRepo,
	}
}

func (s *articleService) CreateArticle(req models.CreateArticleRequest, userID uint) (*models.Article, error) {
	// Process tags save new tags if they don't exist
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

	// Create article first, then version
	if _, err := s.articleRepo.Create(article); err != nil {
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

	// Calculate article tag relationship score
	fmt.Println("Calculating tag relationship score for article ID:", article.ID)
	score := s.CalculateTagRelationshipScore(int(article.ID))
	fmt.Println("Calculated score:", score)

	// Update article with tag relationship score
	err = s.articleRepo.UpdateVersion(version.ID, map[string]interface{}{
		"article_tag_relationship_score": score,
	})
	if err != nil {
		return nil, err
	}

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

	// Delete article versions first
	if err := s.articleVersionRepo.DeleteVersionsByArticleID(id); err != nil {
		return err
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
	version.ArticleTagRelationshipScore = s.CalculateTagRelationshipScore(articleIDint)

	if err := s.articleRepo.CreateVersion(version); err != nil {
		return nil, err
	}

	// Update article's latest version
	if err := s.articleRepo.UpdateFields(articleID, map[string]interface{}{
		"latest_version_id": version.ID,
	}); err != nil {
		return nil, err
	}

	// Update tag usage counts
	s.updateTagUsageCounts()

	return s.articleRepo.GetVersionByID(version.ID)
}

func (s *articleService) UpdateVersionStatus(articleID, versionID uint, status models.VersionStatus, userID uint) error {
	fmt.Println("Updating version status: v1 ", versionID, "to", status, " for article", articleID)
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
			if err != nil {
				return fmt.Errorf("failed to get current published version: %w", err)
			} else {
				if err = s.articleRepo.UpdateVersion(
					currentPublished.ID,
					map[string]interface{}{
						"status": models.StatusArchivedVersion,
					},
				); err != nil {
					return fmt.Errorf("failed to archive current published version: %w", err)
				}
				// 10 updated to archived
			}
		}

		// Set new version as published
		version.Status = status
		now := time.Now()
		version.PublishedAt = &now

		// Update article's published version
		articleFields := map[string]interface{}{
			"published_version_id": versionID,
		}
		if err := s.articleRepo.UpdateFields(articleID, articleFields); err != nil {
			return fmt.Errorf("failed to update article fields: %w", err)
		}

	} else if status == models.StatusArchivedVersion {
		// If archiving the currently published version
		if article.PublishedVersionID != nil && *article.PublishedVersionID == version.ID {
			// This is unpublishing scenario - no published version anymore
			if err := s.articleRepo.ClearPublishedVersionID(article.ID); err != nil {
				return fmt.Errorf("failed to clear published version: %w", err)
			}
		}

		version.Status = status
		// Clear published date when archiving
		version.PublishedAt = nil

	} else {
		// Handle other status changes (draft, etc.)
		version.Status = status

		// If this version was published and now changing to draft, clear article's published reference
		if article.PublishedVersionID != nil && *article.PublishedVersionID == version.ID {
			article.PublishedVersionID = nil
			if err := s.articleRepo.Update(article); err != nil {
				return fmt.Errorf("failed to clear published version reference: %w", err)
			}
		}
	}

	// Update the version
	if err := s.articleRepo.UpdateVersion(versionID, map[string]interface{}{
		"status":       version.Status,
		"published_at": version.PublishedAt,
	}); err != nil {
		return fmt.Errorf("failed to update version: %w", err)
	}

	// Update tag usage counts after status change
	s.updateTagUsageCounts()

	return nil
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

func (s *articleService) CalculateTagRelationshipScore(articleID int) float64 {
	// 1. Ambil semua tag dari artikel ini
	tags, err := s.articleRepo.GetTagsForArticle(articleID)
	if err != nil {
		fmt.Println("Error getting tags for article:", err)
		return 0.0
	}
	if len(tags) < 2 {
		return 0.0
	}
	fmt.Println("Tags for article:", tags)

	// 2. Ambil total artikel
	totalArticles, err := s.articleRepo.GetTotalArticleCount()
	if err != nil {
		fmt.Println("Error getting total article count:", err)
		return 0.0
	}
	totalArticlesF := float64(totalArticles)
	fmt.Printf("Total articles: %d (float: %.0f)\n", totalArticles, totalArticlesF)

	// 3. Ambil frekuensi semua tag
	tagFreq, err := s.articleRepo.GetTagFrequencies(tags)
	if err != nil {
		fmt.Println("Error getting tag frequencies:", err)
		return 0.0
	}
	fmt.Println("Tag frequencies:", tagFreq)

	// 4. Ambil co-occurrence semua pasangan tag
	coOccurMap, err := s.articleRepo.GetTagPairCoOccurrences(tags)
	if err != nil {
		fmt.Println("Error getting tag pair co-occurrences:", err)
		return 0.0
	}
	fmt.Println("Co-occurrence map:", coOccurMap)

	// 5. Hitung skor
	scoreSum := 0.0
	pairCount := 0

	// helper untuk urutkan tag sesuai LEAST/GREATEST
	minString := func(a, b string) string {
		if a < b {
			return a
		}
		return b
	}
	maxString := func(a, b string) string {
		if a > b {
			return a
		}
		return b
	}

	for i := 0; i < len(tags)-1; i++ {
		for j := i + 1; j < len(tags); j++ {
			tag1 := tags[i]
			tag2 := tags[j]

			// Pastikan key cocok
			key := fmt.Sprintf("%s|%s", minString(tag1, tag2), maxString(tag1, tag2))
			freqA := float64(tagFreq[tag1])
			freqB := float64(tagFreq[tag2])
			coOccur := float64(coOccurMap[key])

			// Debug semua data
			fmt.Printf("Pair: %-10s & %-10s | freqA=%-4.0f freqB=%-4.0f coOccur=%-4.0f\n",
				tag1, tag2, freqA, freqB, coOccur)

			if freqA == 0 || freqB == 0 || coOccur == 0 {
				continue
			}

			pTag1 := freqA / totalArticlesF
			pTag2 := freqB / totalArticlesF
			pBoth := coOccur / totalArticlesF

			pmi := math.Log(pBoth / (pTag1 * pTag2))

			// Kalau mau Positive PMI aktifkan ini:
			// if pmi < 0 { pmi = 0 }

			fmt.Printf("  -> pTag1=%.4f pTag2=%.4f pBoth=%.4f PMI=%.4f\n", pTag1, pTag2, pBoth, pmi)

			scoreSum += pmi
			pairCount++
		}
	}

	if pairCount == 0 {
		return 0.0
	}
	fmt.Printf("Pair count: %d | Score sum: %.4f | Final score: %.4f\n", pairCount, scoreSum, scoreSum/float64(pairCount))
	return scoreSum / float64(pairCount)
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
	// Ambil usage count dari artikel published
	tagCounts, err := s.articleRepo.CountArticlesByTag()
	if err != nil {
		return
	}

	// Ambil semua tag
	allTags, err := s.tagRepo.GetAll()
	if err != nil {
		return
	}

	const decayFactor = 7.0
	var tagsToUpdate []models.Tag

	for i := range allTags {
		currentTag := &allTags[i]

		// Hitung usage count baru
		newCount := 0
		if count, exists := tagCounts[currentTag.ID]; exists {
			newCount = count
		}

		// Hitung umur dari last update
		ageInDays := time.Since(currentTag.UpdatedAt).Hours() / 24
		newTrendingScore := float64(newCount) * math.Exp(-ageInDays/decayFactor)

		// Cek perubahan
		if newCount != currentTag.UsageCount || !floatAlmostEqual(newTrendingScore, currentTag.TrendingScore) {
			currentTag.UsageCount = newCount
			currentTag.TrendingScore = newTrendingScore

			// Reset updated_at kalau ada usage baru
			if newCount > currentTag.UsageCount {
				currentTag.UpdatedAt = time.Now()
			}

			tagsToUpdate = append(tagsToUpdate, *currentTag)
		}
	}

	if len(tagsToUpdate) > 0 {
		_ = s.tagRepo.BulkUpdate(tagsToUpdate)
	}
}

// floatAlmostEqual membandingkan float agar tidak sensitif terhadap perbedaan kecil
func floatAlmostEqual(a, b float64) bool {
	const epsilon = 0.000001
	return math.Abs(a-b) < epsilon
}
