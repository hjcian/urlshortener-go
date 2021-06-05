package controllers

import (
	"errors"
	"fmt"
	"goshorturl/models"
	"goshorturl/urlshortener"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const expireAtLayout = "2006-01-02T15:04:05Z"

type uploadReqData struct {
	Url      string `json:"url"`
	ExpireAt string `json:"expireAt"`
	expireAt time.Time
}

// parseAndValidate parses the expireAt and stores result if parsing successful.
//
// Also validates the requested data should be valid.
//
// Return non-nil error if validation failed.
func (u *uploadReqData) parseAndValidate() (err error) {
	if _, err = url.ParseRequestURI(u.Url); err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	u.expireAt, err = time.Parse(expireAtLayout, u.ExpireAt)
	if err != nil {
		return err
	}
	now := time.Now()
	if u.expireAt.Before(now) {
		return errors.New("uploaded URL has already expired")
	}
	return nil
}

type UrlController struct {
	DB             *gorm.DB
	Log            *zap.Logger
	RedirectOrigin string
}

func (u UrlController) Upload(c *gin.Context) {
	var req uploadReqData
	err := c.BindJSON(&req)
	if err != nil {
		u.Log.Warn("invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := req.parseAndValidate(); err != nil {
		u.Log.Warn("invalid upload data", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid upload data"})
		return
	}

	key := urlshortener.Get(req.Url)
	u.Log.Debug("get key", zap.String("key", key))
	urlEntry := models.Url{
		Key:       key,
		Url:       req.Url,
		ExpiredAt: req.expireAt,
	}
	if err := u.DB.Create(&urlEntry).Error; err != nil {
		u.Log.Error("upload error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal upload error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       key,
		"shortUrl": fmt.Sprintf("%s/%s", u.RedirectOrigin, key),
	})
}

func (u UrlController) Delete(c *gin.Context) {
	urlID := c.Param("url_id")
	if err := urlshortener.Validate(urlID); err != nil {
		u.Log.Warn("invalid id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	res := u.DB.Debug().Delete(&models.Url{Key: urlID})
	if res.Error != nil {
		u.Log.Error("delete error", zap.Error(res.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete error"})
		return
	}
	if res.RowsAffected != 1 {
		u.Log.Warn("id not exists", zap.String("id", urlID))
		c.JSON(http.StatusNotFound, gin.H{"error": "id not exists"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (u UrlController) Redirect(c *gin.Context) {
	urlID := c.Param("url_id")
	if err := urlshortener.Validate(urlID); err != nil {
		u.Log.Warn("invalid id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var result models.Url
	if err := u.DB.Where(
		// NOTE: will use `"urls"."deleted_at" IS NULL` to filter the deleted record
		"key = ? AND expired_at > ?",
		urlID, time.Now(),
	).Take(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// TODO: also cache empty data to mitigate cache penetration
			c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "redirect error"})
		return
	}
	// TODO: cache data to mitigate cache penetration
	c.Redirect(http.StatusMovedPermanently, result.Url)
}
