package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"nutrai/internal/config"
	"nutrai/internal/handler"
	"nutrai/internal/middleware"
	"nutrai/internal/n8n"
	"nutrai/internal/repository"
	"nutrai/internal/service"
)

func New(cfg *config.Config, db *gorm.DB) *gin.Engine {
	gin.SetMode(cfg.GinMode)
	r := gin.New()
	r.MaxMultipartMemory = 8 << 20
	r.Use(gin.Recovery(), gin.Logger(), middleware.CORS(cfg.AllowedOrigins))

	// --- wiring (dependency injection manual) ---
	repo := repository.New(db)
	n8nClient := n8n.NewClient(cfg.N8NSecret, cfg.N8NTimeout, cfg.N8NMaxRetries)

	authSvc := service.NewAuthService(repo)
	profileSvc := service.NewProfileService(repo)
	logSvc := service.NewLogService(repo, cfg.Location)
	foodSvc := service.NewFoodService(repo, profileSvc, logSvc, cfg.Location, cfg.FoodImageBaseURL)
	aiSvc := service.NewAIService(n8nClient, cfg, profileSvc, logSvc, foodSvc)

	authH := handler.NewAuthHandler(authSvc)
	profileH := handler.NewProfileHandler(profileSvc)
	logH := handler.NewLogHandler(logSvc)
	foodH := handler.NewFoodHandler(foodSvc)
	aiH := handler.NewAIHandler(aiSvc, cfg.MaxUploadBytes)

	r.Static("/static", "./static") // gambar menu: static/foods/<slug>.jpg

	r.GET("/healthz", func(c *gin.Context) {
		if err := repo.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "db_unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", authH.Register)

	secured := v1.Group("")
	secured.Use(middleware.APIKeyAuth(repo))
	{
		secured.POST("/auth/generate-key", authH.GenerateKey)

		secured.GET("/profile", profileH.Get)
		secured.PUT("/profile", profileH.Update)

		secured.GET("/logs/today", logH.Today)
		secured.POST("/logs/meal", logH.AddMeal)
		secured.POST("/logs/water", logH.AddWater)
		secured.GET("/logs/history", logH.History)

		secured.GET("/favorites", logH.ListFavorites)
		secured.POST("/favorites", logH.AddFavorite)

		secured.GET("/foods", foodH.Search)
		secured.GET("/foods/:id", foodH.Detail)
		secured.GET("/suggestions", foodH.Suggestions)
		secured.GET("/ingredients", foodH.Ingredients)

		secured.POST("/ai/scan", aiH.Scan)
		secured.POST("/ai/chat", aiH.Chat)
	}
	return r
}
