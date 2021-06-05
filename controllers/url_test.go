package controllers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestUrlController_Upload(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	validExpireTimeStr := time.Now().Add(time.Hour).Format(expireAtLayout)
	expiredTimeStr := time.Now().Add(-24 * time.Hour).Format(expireAtLayout)

	tests := []struct {
		name               string
		reqJSON            string
		expectedStatusCode int
	}{
		{
			"invalid url",
			fmt.Sprintf(`{"url": "foobar", "expireAt": "%s"}`, validExpireTimeStr),
			http.StatusBadRequest,
		},
		{
			"empty url",
			fmt.Sprintf(`{"url": "", "expireAt": "%s"}`, validExpireTimeStr),
			http.StatusBadRequest,
		},
		{
			"no url field",
			fmt.Sprintf(`{"expireAt": "%s"}`, validExpireTimeStr),
			http.StatusBadRequest,
		},
		{
			"invalid time format",
			`{"url": "http://example.com", "expireAt": "foobar"}}`,
			http.StatusBadRequest,
		},
		{
			"upload an url with expired time",
			fmt.Sprintf(`{"url": "http://example.com", "expireAt": "%s"}`, expiredTimeStr),
			http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(r)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.reqJSON))

			u := UrlController{
				DB:             nil,
				Log:            logger,
				RedirectOrigin: "",
			}
			u.Upload(ctx)
			assert.Equal(t, tt.expectedStatusCode, r.Code)
		})
	}
}

func TestUrlController_Redirect(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name               string
		id                 string
		expectedStatusCode int
	}{
		// {
		// 	"redirect OK",
		// 	"qwerty",
		// 	http.StatusMovedPermanently,
		// },
		{
			"empty id",
			"",
			http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(r)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Params = []gin.Param{{Key: "url_id", Value: tt.id}}

			u := UrlController{
				DB:             nil,
				Log:            logger,
				RedirectOrigin: "",
			}
			u.Redirect(c)
			assert.Equal(t, tt.expectedStatusCode, r.Code)
		})
	}
}
