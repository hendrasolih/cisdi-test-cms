package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
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
	os.Setenv("DB_USER", "myuser")
	os.Setenv("DB_PASSWORD", "mypassword")
	os.Setenv("DB_NAME", "cms_test_db")
	os.Setenv("JWT_SECRET", "test-secret")

	// Initialize test database
	dsn := "host=localhost port=5432 user=myuser password=mypassword dbname=cms_test_db sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		suite.T().Fatal("Failed to connect to test database:", err)
	}

	suite.db = db

	if err := RunSQLFile(db, "../migration/init.sql"); err != nil {
		log.Fatal("Failed migrate users:", err)
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
	articleVersionRepo := repositories.NewArticleVersionRepository(suite.db)

	// Initialize services
	authService := services.NewAuthService(userRepo)
	articleService := services.NewArticleService(articleRepo, tagRepo, articleVersionRepo)
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
	registerPayload := models.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     models.RoleAdmin,
	}

	body, _ := json.Marshal(registerPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code) // karena register mengembalikan 200
	// sesuaikan expected status

	type RegisterResponse struct {
		Code        int                 `json:"code"`
		CodeMessage string              `json:"code_message"`
		CodeType    string              `json:"code_type"`
		Data        models.AuthResponse `json:"data"`
	}

	var registerResponse RegisterResponse
	err := json.Unmarshal(w.Body.Bytes(), &registerResponse)
	suite.NoError(err)

	suite.token = registerResponse.Data.Token
	suite.userID = registerResponse.Data.User.ID
	fmt.Println("Registered user ID:", suite.userID)
}

func (suite *IntegrationTestSuite) TestAuthFlow() {
	loginPayload := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	body, _ := json.Marshal(loginPayload)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	type LoginResponse struct {
		Code        int                 `json:"code"`
		CodeMessage string              `json:"code_message"`
		CodeType    string              `json:"code_type"`
		Data        models.AuthResponse `json:"data"`
	}

	var loginResp LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &loginResp)
	suite.NoError(err)

	response := loginResp.Data

	suite.NotEmpty(response.Token)
	suite.Equal("testuser", response.User.Username)
}

func (suite *IntegrationTestSuite) TestGetProfile() {
	req := httptest.NewRequest("GET", "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	type ProfileResponse struct {
		Code        int         `json:"code"`
		CodeMessage string      `json:"code_message"`
		CodeType    string      `json:"code_type"`
		Data        models.User `json:"data"`
	}

	var profileResp ProfileResponse
	err := json.Unmarshal(w.Body.Bytes(), &profileResp)
	suite.NoError(err)

	user := profileResp.Data
	suite.Equal("testuser", user.Username)
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

	suite.Equal(http.StatusOK, w.Code)

	type CreateArticleResponse struct {
		Code        int            `json:"code"`
		CodeMessage string         `json:"code_message"`
		CodeType    string         `json:"code_type"`
		Data        models.Article `json:"data"`
	}

	var createResp CreateArticleResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	suite.NoError(err)
	article := createResp.Data

	suite.Equal("Test Article", article.Title)
	suite.Equal(suite.userID, article.AuthorID)

	// Get article
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/articles/%d", article.ID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	type GetArticleResponse struct {
		Code        int            `json:"code"`
		CodeMessage string         `json:"code_message"`
		CodeType    string         `json:"code_type"`
		Data        models.Article `json:"data"`
	}

	var getResp GetArticleResponse
	err = json.Unmarshal(w.Body.Bytes(), &getResp)
	suite.NoError(err)
	retrievedArticle := getResp.Data

	suite.Equal(article.ID, retrievedArticle.ID)
	suite.Equal("Test Article", retrievedArticle.Title)
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

	type CreateArticleResponse struct {
		Code        int            `json:"code"`
		CodeMessage string         `json:"code_message"`
		CodeType    string         `json:"code_type"`
		Data        models.Article `json:"data"`
	}

	var createResp CreateArticleResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	suite.NoError(err)
	article := createResp.Data

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

	type CreateVersionResponse struct {
		Code        int                   `json:"code"`
		CodeMessage string                `json:"code_message"`
		CodeType    string                `json:"code_type"`
		Data        models.ArticleVersion `json:"data"`
	}

	var versionResp CreateVersionResponse
	err = json.Unmarshal(w.Body.Bytes(), &versionResp)
	suite.NoError(err)
	version := versionResp.Data

	suite.Equal(http.StatusOK, w.Code)
	suite.Equal(2, version.VersionNumber)
	suite.Equal("Updated Article", version.Title)

	// Get versions
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/articles/%d/versions", article.ID), nil)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	type GetVersionsResponse struct {
		Code        int                     `json:"code"`
		CodeMessage string                  `json:"code_message"`
		CodeType    string                  `json:"code_type"`
		Data        []models.ArticleVersion `json:"data"`
	}

	var versionsResp GetVersionsResponse
	err = json.Unmarshal(w.Body.Bytes(), &versionsResp)
	suite.NoError(err)
	versions := versionsResp.Data

	suite.Equal(http.StatusOK, w.Code)
	suite.Len(versions, 2)
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

	type CreateArticleResponse struct {
		Code        int            `json:"code"`
		CodeMessage string         `json:"code_message"`
		CodeType    string         `json:"code_type"`
		Data        models.Article `json:"data"`
	}

	var createResp CreateArticleResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	suite.NoError(err)
	article := createResp.Data

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

	suite.Equal(http.StatusOK, w.Code)

	// Check if article is now accessible via public API
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/public/articles/%d", article.ID), nil)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
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

	suite.Equal(http.StatusOK, w.Code)

	var createResp struct {
		Code        int        `json:"code"`
		CodeMessage string     `json:"code_message"`
		CodeType    string     `json:"code_type"`
		Data        models.Tag `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	suite.NoError(err)
	suite.Equal("manual-tag", createResp.Data.Name)

	// Get all tags
	req = httptest.NewRequest("GET", "/api/v1/tags", nil)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var getTagsResp struct {
		Code        int          `json:"code"`
		CodeMessage string       `json:"code_message"`
		CodeType    string       `json:"code_type"`
		Data        []models.Tag `json:"data"`
	}

	err = json.Unmarshal(w.Body.Bytes(), &getTagsResp)
	suite.NoError(err)
	suite.GreaterOrEqual(len(getTagsResp.Data), 1)
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

		suite.Equal(http.StatusCreated, w.Code)
	}

	// Get articles to check scores
	req := httptest.NewRequest("GET", "/api/v1/articles", nil)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)

	var response struct {
		Articles []models.Article `json:"articles"`
		Total    int64            `json:"total"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Len(response.Articles, 3)

	// Check that articles have relationship scores calculated
	for _, article := range response.Articles {
		if len(article.LatestVersion.Tags) >= 2 {
			// Articles with multiple tags should have some relationship score
			suite.GreaterOrEqual(article.LatestVersion.ArticleTagRelationshipScore, 0.0)
		}
	}
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func RunSQLFile(db *gorm.DB, filepath string) error {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	return db.Exec(string(content)).Error
}
