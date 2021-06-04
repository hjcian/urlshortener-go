package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type UrlController struct{}

func (u UrlController) Upload(c *gin.Context) {
	c.String(http.StatusOK, "upload")
}

func (u UrlController) Delete(c *gin.Context) {
	urlID := c.Param("url_id")
	c.String(http.StatusOK, "delete %s", urlID)
}

func (u UrlController) Redirect(c *gin.Context) {
	urlID := c.Param("url_id")
	c.String(http.StatusOK, "redirect %s", urlID)
}
