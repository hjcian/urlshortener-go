package controllers

import (
	"errors"
	"fmt"
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
	// urlEntry := models.Url{
	// 	Key:       key,
	// 	Url:       req.Url,
	// 	ExpiredAt: req.expireAt,
	// }
	// if err := u.DB.Create(&urlEntry).Error; err != nil {
	// 	// TODO
	// }

	c.JSON(http.StatusOK, gin.H{
		"id":       key,
		"shortUrl": fmt.Sprintf("%s/%s", u.RedirectOrigin, key),
	})
	// u.DB.Create()
	// gen key
	// insert (key, url, expireAt, createAt, UpdateAt, deleteAt)
	// if conflict, just re-generate key (shift one letter) and insert again
}

func (u UrlController) Delete(c *gin.Context) {
	urlID := c.Param("url_id")
	c.String(http.StatusOK, "delete %s", urlID)
	// update record
}

func (u UrlController) Redirect(c *gin.Context) {
	urlID := c.Param("url_id")
	c.String(http.StatusOK, "redirect %s", urlID)
	// select url from db where id=id AND expireAt > now AND deleteAt != NULL
}
