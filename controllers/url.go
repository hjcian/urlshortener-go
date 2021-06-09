package controllers

import (
	"context"
	"errors"
	"fmt"
	"goshorturl/idgenerator"
	"goshorturl/repository"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const expireAtLayout = "2006-01-02T15:04:05Z"

type uploadReqData struct {
	Url         string `json:"url"`
	ExpireAtStr string `json:"expireAt"`
	expireAt    time.Time
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

	u.expireAt, err = time.Parse(expireAtLayout, u.ExpireAtStr)
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
	DB             repository.Repository
	Log            *zap.Logger
	IDGenerator    idgenerator.IDGenerator
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

	id, err := u.IDGenerator.Get(context.Background(), req.Url, req.expireAt)
	if err != nil {
		u.Log.Error("upload error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal upload error"})
		return
	}
	// id := idgenerator.Generate(req.Url)
	// if err := u.DB.Create(context.Background(), id, req.Url, req.expireAt); err != nil {
	// 	u.Log.Error("upload error", zap.Error(err))
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal upload error"})
	// 	return
	// }

	c.JSON(http.StatusOK, gin.H{
		"id":       id,
		"shortUrl": fmt.Sprintf("%s/%s", u.RedirectOrigin, id),
	})
}

func (u UrlController) Delete(c *gin.Context) {
	urlID := c.Param("url_id")
	if err := idgenerator.Validate(urlID); err != nil {
		u.Log.Warn("invalid id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := u.DB.Delete(context.Background(), urlID); err != nil {
		if err == repository.ErrRecordNotFound {
			u.Log.Warn("id not exists", zap.String("id", urlID))
			c.JSON(http.StatusNotFound, gin.H{"error": "id not exists"})
			return
		}
		u.Log.Error("delete error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete error"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (u UrlController) Redirect(c *gin.Context) {
	urlID := c.Param("url_id")
	if err := idgenerator.Validate(urlID); err != nil {
		u.Log.Warn("invalid id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	oriUrl, err := u.DB.Get(context.Background(), urlID)
	if err != nil {
		if err == repository.ErrRecordNotFound {
			u.Log.Warn("record not found", zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
			return
		}
		u.Log.Error("redirect error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "redirect error"})
		return
	}
	c.Redirect(http.StatusMovedPermanently, oriUrl)
}
