package main

import (
	"log"
	"net/http"
	"os"

	"cisdi-test-cms/config"
	"cisdi-test-cms/handlers"
	"cisdi-test-cms/middleware"
	"cisdi-test-cms/repositories"
	"cisdi-test-cms/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	db := config.InitDB()

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	articleRepo := repositories.NewArticleRepository(db)
	tagRepo := repositories.NewTagRepository(db)
	articleVersionRepo := repositories.NewArticleVersionRepository(db)

	// Initialize services
	authService := services.NewAuthService(userRepo)
	articleService := services.NewArticleService(articleRepo, tagRepo, articleVersionRepo)
	tagService := services.NewTagService(tagRepo, articleRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	articleHandler := handlers.NewArticleHandler(articleService)
	tagHandler := handlers.NewTagHandler(tagService)

	// Setup router
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Protected routes
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			// Profile
			protected.GET("/profile", authHandler.GetProfile)

			// Articles
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

			// Tags
			tags := protected.Group("/tags")
			{
				tags.POST("", tagHandler.CreateTag)
				tags.GET("", tagHandler.GetTags)
				tags.GET("/:id", tagHandler.GetTag)
			}
		}

		// Public article routes (published only)
		public := v1.Group("/public")
		{
			public.GET("/articles", articleHandler.GetPublicArticles)
			public.GET("/articles/:id", articleHandler.GetPublicArticle)
		}
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
