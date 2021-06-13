package server

import (
	"context"
	"goshorturl/controllers"
	"goshorturl/idgenerator"
	"goshorturl/repository"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	defaultTimeout = 30 * time.Second
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

	router.POST("/api/v1/urls", withTimeout(url.Upload, defaultTimeout))
	router.DELETE("/api/v1/urls/:url_id", withTimeout(url.Delete, defaultTimeout))
	router.GET("/:url_id", withTimeout(url.Redirect, defaultTimeout))

	return router
}

func withTimeout(handler gin.HandlerFunc, timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		ch := make(chan struct{}, 1)
		go func() {
			defer func() {
				_ = gin.Recovery()
			}()
			handler(c)
			ch <- struct{}{}
		}()

		select {
		case <-ch:
			c.Next()
		case <-time.After(timeout):
			c.AbortWithStatus(http.StatusRequestTimeout)
			c.String(http.StatusRequestTimeout, http.StatusText(http.StatusRequestTimeout))
			return
		}
	}
}
