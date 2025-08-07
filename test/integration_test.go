package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"cisdi-test-cms/handlers"
	"cisdi-test-cms/middleware"
	"cisdi-test-cms/models"
	"cisdi-test-cms/repositories"
	"cisdi-test-cms/services"
)

type IntegrationTestSuite struct {
	suite.Suite
	db     *gorm.DB
	router *gin.Engine
	token  string
	userID uint
}

func (suite *IntegrationTestSuite) SetupSuite() {
	// Set test environment
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "password")
	os.Setenv("DB_NAME", "cms_test_db")
	os.Setenv("JWT_SECRET", "test-secret")

	// Initialize test database
	dsn := "host=localhost port=5432 user=postgres password=password dbname=cms_test_db sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		suite.T().Fatal("Failed to connect to test database:", err)
	}

	suite.db = db

	// Migrate tables
	err = db.AutoMigrate(
		&models.User{},
		&models.Article{},
		&models.ArticleVersion{},
		&models.Tag{},
		&models.ArticleVersionTag{},
	)
	if err != nil {
		suite.T().Fatal("Failed to migrate test database:", err)
	}

	// Setup router
	suite.setupRouter()
}

func (suite *IntegrationTestSuite) setupRouter() {
	gin.SetMode(gin.TestMode)

	// Initialize repositories
	userRepo := repositories.NewUserRepository(suite.db)
	articleRepo := repositories.NewArticleRepository(suite.db)
	tagRepo := repositories.NewTagRepository(suite.db)

	// Initialize services
	authService := services.NewAuthService(userRepo)
	articleService := services.NewArticleService(articleRepo, tagRepo)
	tagService := services.NewTagService(tagRepo, articleRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	articleHandler := handlers.NewArticleHandler(articleService)
	tagHandler := handlers.NewTagHandler(tagService)

	// Setup router
	router := gin.New()

	v1 := router.Group("/api/v1")
	{
		// Auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Protected routes
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/profile", authHandler.GetProfile)

			articles := protected.Group("/articles")
			{
				articles.POST("", articleHandler.CreateArticle)
				articles.GET("", articleHandler.GetArticles)
				articles.GET("/:id", articleHandler.GetArticle)
				articles.DELETE("/:id", articleHandler.DeleteArticle)
				articles.POST("/:id/versions", articleHandler.CreateArticleVersion)
				articles.PUT("/:id/versions/:version_id/status", articleHandler.UpdateVersionStatus)
				articles.GET("/:id/versions", articleHandler.GetArticleVersions)
				articles.GET("/:id/versions/:version_id", articleHandler.GetArticleVersion)
			}

			tags := protected.Group("/tags")
			{
				tags.POST("", tagHandler.CreateTag)
				tags.GET("", tagHandler.GetTags)
				tags.GET("/:id", tagHandler.GetTag)
			}
		}

		// Public routes
		public := v1.Group("/public")
		{
			public.GET("/articles", articleHandler.GetPublicArticles)
			public.GET("/articles/:id", articleHandler.GetPublicArticle)
		}
	}

	suite.router = router
}

func (suite *IntegrationTestSuite) TearDownSuite() {
	// Clean up test database
	suite.db.Exec("DROP TABLE IF EXISTS article_version_tags")
	suite.db.Exec("DROP TABLE IF EXISTS article_versions")
	suite.db.Exec("DROP TABLE IF EXISTS articles")
	suite.db.Exec("DROP TABLE IF EXISTS tags")
	suite.db.Exec("DROP TABLE IF EXISTS users")
}

func (suite *IntegrationTestSuite) SetupTest() {
	// Clean all tables before each test
	suite.db.Exec("TRUNCATE TABLE article_version_tags RESTART IDENTITY CASCADE")
	suite.db.Exec("TRUNCATE TABLE article_versions RESTART IDENTITY CASCADE")
	suite.db.Exec("TRUNCATE TABLE articles RESTART IDENTITY CASCADE")
	suite.db.Exec("TRUNCATE TABLE tags RESTART IDENTITY CASCADE")
	suite.db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")

	// Register and login a test user
	suite.registerAndLoginTestUser()
}

func (suite *IntegrationTestSuite) registerAndLoginTestUser() {
	// Register user
	registerPayload := models.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     models.RoleWriter,
	}

	body, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var registerResponse models.AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &registerResponse)
	assert.NoError(suite.T(), err)

	suite.token = registerResponse.Token
	suite.userID = registerResponse.User.ID
}

func (suite *IntegrationTestSuite) TestAuthFlow() {
	// Test login
	loginPayload := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	body, _ := json.Marshal(loginPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), response.Token)
	assert.Equal(suite.T(), "testuser", response.User.Username)
}

func (suite *IntegrationTestSuite) TestGetProfile() {
	req := httptest.NewRequest("GET", "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var user models.User
	err := json.Unmarshal(w.Body.Bytes(), &user)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "testuser", user.Username)
}

func (suite *IntegrationTestSuite) TestCreateAndGetArticle() {
	// Create article
	createPayload := models.CreateArticleRequest{
		Title:   "Test Article",
		Content: "<p>This is test content</p>",
		Tags:    []string{"golang", "api", "test"},
	}

	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest("POST", "/api/v1/articles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var article models.Article
	err := json.Unmarshal(w.Body.Bytes(), &article)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Test Article", article.Title)
	assert.Equal(suite.T(), suite.userID, article.AuthorID)

	// Get article
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/articles/%d", article.ID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var retrievedArticle models.Article
	err = json.Unmarshal(w.Body.Bytes(), &retrievedArticle)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), article.ID, retrievedArticle.ID)
	assert.Equal(suite.T(), "Test Article", retrievedArticle.Title)
}

func (suite *IntegrationTestSuite) TestArticleVersioning() {
	// Create article
	createPayload := models.CreateArticleRequest{
		Title:   "Versioned Article",
		Content: "<p>Original content</p>",
		Tags:    []string{"version", "test"},
	}

	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest("POST", "/api/v1/articles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	var article models.Article
	json.Unmarshal(w.Body.Bytes(), &article)

	// Create new version
	versionPayload := models.CreateArticleVersionRequest{
		Title:   "Updated Article",
		Content: "<p>Updated content</p>",
		Tags:    []string{"version", "updated"},
	}

	body, _ = json.Marshal(versionPayload)
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/articles/%d/versions", article.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var version models.ArticleVersion
	err := json.Unmarshal(w.Body.Bytes(), &version)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, version.VersionNumber)
	assert.Equal(suite.T(), "Updated Article", version.Title)

	// Get versions
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/articles/%d/versions", article.ID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var versions []models.ArticleVersion
	err = json.Unmarshal(w.Body.Bytes(), &versions)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), versions, 2)
}

func (suite *IntegrationTestSuite) TestPublishArticle() {
	// Create article
	createPayload := models.CreateArticleRequest{
		Title:   "Article to Publish",
		Content: "<p>Content to publish</p>",
		Tags:    []string{"publish", "test"},
	}

	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest("POST", "/api/v1/articles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	var article models.Article
	json.Unmarshal(w.Body.Bytes(), &article)

	// Publish version
	publishPayload := models.UpdateVersionStatusRequest{
		Status: models.StatusPublished,
	}

	body, _ = json.Marshal(publishPayload)
	req = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/articles/%d/versions/%d/status", article.ID, article.LatestVersion.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Check if article is now accessible via public API
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/public/articles/%d", article.ID), nil)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *IntegrationTestSuite) TestTagManagement() {
	// Create tag
	createPayload := models.CreateTagRequest{
		Name: "manual-tag",
	}

	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest("POST", "/api/v1/tags", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var tag models.Tag
	err := json.Unmarshal(w.Body.Bytes(), &tag)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "manual-tag", tag.Name)

	// Get all tags
	req = httptest.NewRequest("GET", "/api/v1/tags", nil)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var tags []models.Tag
	err = json.Unmarshal(w.Body.Bytes(), &tags)
	assert.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(tags), 1)
}

func (suite *IntegrationTestSuite) TestArticleTagRelationshipScore() {
	// Create articles with common tags to test relationship scoring
	articles := []models.CreateArticleRequest{
		{
			Title:   "Go Programming",
			Content: "<p>About Go programming</p>",
			Tags:    []string{"golang", "programming", "backend"},
		},
		{
			Title:   "API Development",
			Content: "<p>About API development</p>",
			Tags:    []string{"api", "golang", "rest"},
		},
		{
			Title:   "Database Design",
			Content: "<p>About database design</p>",
			Tags:    []string{"database", "postgresql", "design"},
		},
	}

	for _, articleReq := range articles {
		body, _ := json.Marshal(articleReq)
		req := httptest.NewRequest("POST", "/api/v1/articles", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+suite.token)

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusCreated, w.Code)
	}

	// Get articles to check scores
	req := httptest.NewRequest("GET", "/api/v1/articles", nil)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response struct {
		Articles []models.Article `json:"articles"`
		Total    int64            `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response.Articles, 3)

	// Check that articles have relationship scores calculated
	for _, article := range response.Articles {
		if len(article.LatestVersion.Tags) >= 2 {
			// Articles with multiple tags should have some relationship score
			assert.GreaterOrEqual(suite.T(), article.LatestVersion.ArticleTagRelationshipScore, 0.0)
		}
	}
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
