package server

import (
	"goshorturl/controllers"
	"goshorturl/idgenerator"
	"goshorturl/repository"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewRouter(db repository.Repository, idGenerator idgenerator.IDGenerator, logger *zap.Logger, redirectOrigin string) *gin.Engine {
	router := gin.Default()
	router.HandleMethodNotAllowed = true

	health := new(controllers.HealthController)
	router.GET("/health", health.Status)

	url := controllers.UrlController{
		DB:             db,
		Log:            logger,
		IDGenerator:    idGenerator,
		RedirectOrigin: redirectOrigin,
	}
	router.POST("/api/v1/urls", url.Upload)
	router.DELETE("/api/v1/urls/:url_id", url.Delete)
	router.GET("/:url_id", url.Redirect)

	return router
}
