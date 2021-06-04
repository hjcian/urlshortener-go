package server

import (
	"goshorturl/controllers"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	router := gin.Default()
	router.HandleMethodNotAllowed = true

	health := new(controllers.HealthController)
	router.GET("/health", health.Status)

	url := new(controllers.UrlController)
	router.POST("/api/v1/urls", url.Upload)
	router.DELETE("/api/v1/urls/:url_id", url.Delete)
	router.GET("/:url_id", url.Redirect)

	return router
}

func Init() {
	r := NewRouter()
	r.Run(":8080")
}
