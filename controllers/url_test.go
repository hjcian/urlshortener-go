package controllers

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type anyExpireTime struct{}

func (a anyExpireTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

func getMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	require.NoError(t, err)
	return gormDB, mock
}
func TestUrlController_Upload(t *testing.T) {
	gin.SetMode(gin.TestMode)
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

func TestUrlController_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()

	injectMock := func(mock sqlmock.Sqlmock, id string, result driver.Result, wantSQLError bool) {
		mock.ExpectBegin()
		exec := mock.ExpectExec(`UPDATE "urls" SET (.+) WHERE "urls"."key" = ?`)
		if !wantSQLError {
			exec.WithArgs(anyExpireTime{}, id).
				WillReturnResult(result)
		} else {
			exec.WillReturnError(errors.New("db internal error"))
		}
		mock.ExpectCommit()
	}

	tests := []struct {
		name               string
		id                 string
		sqlResult          driver.Result
		wantSQLError       bool
		expectedStatusCode int
	}{
		{
			"delete ok",
			"okokok",
			sqlmock.NewResult(1, 1),
			false,
			http.StatusNoContent,
		},
		{
			"not found id",
			"noidno",
			sqlmock.NewResult(1, 0),
			false,
			http.StatusNotFound,
		},
		{
			"empty id",
			"",
			sqlmock.NewResult(1, 0),
			false,
			http.StatusBadRequest,
		},
		{
			"internal db error",
			"okokok",
			nil,
			true,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(r)
			c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
			c.Params = []gin.Param{{Key: "url_id", Value: tt.id}}

			gormDB, mock := getMockDB(t)
			injectMock(mock, tt.id, tt.sqlResult, tt.wantSQLError)

			u := UrlController{
				DB:             gormDB,
				Log:            logger,
				RedirectOrigin: "",
			}
			u.Delete(c)
			assert.Equal(t, tt.expectedStatusCode, r.Code)
		})
	}
}

func TestUrlController_Redirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
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
