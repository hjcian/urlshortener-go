package server

import (
	"goshorturl/controllers"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, logger *zap.Logger, redirectOrigin string) *gin.Engine {
	router := gin.Default()
	router.HandleMethodNotAllowed = true

	health := new(controllers.HealthController)
	router.GET("/health", health.Status)

	url := controllers.UrlController{
		DB:             db,
		Log:            logger,
		RedirectOrigin: redirectOrigin,
	}
	router.POST("/api/v1/urls", url.Upload)
	router.DELETE("/api/v1/urls/:url_id", url.Delete)
	router.GET("/:url_id", url.Redirect)

	return router
}
