package controllers

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"goshorturl/idgenerator"
	"goshorturl/repository"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	errInternalDBError = errors.New("internal db error raised by test")
)

type anyExpireTime struct{}

func (a anyExpireTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

type anyValidID struct{}

func (a anyValidID) Match(v driver.Value) bool {
	id, ok := v.(string)
	err := idgenerator.Validate(id)
	return ok && (err == nil)
}

func getMockDB(t *testing.T) (repository.Repository, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	assert.NoError(t, err)

	repo, err := repository.NewPGRepoForTestWith(
		postgres.New(postgres.Config{Conn: sqlDB}),
		gorm.Config{
			Logger: logger.Default.LogMode(logger.Info), // display SQL statement for debugging
		},
	)
	assert.NoError(t, err)

	return repo, mock
}

func TestUrlController_Upload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()

	redirectOrigin := "http://example.com"
	validExpireTime := time.Now().UTC().Add(24 * time.Hour)
	expiredTime := time.Now().UTC().Add(-24 * time.Hour)

	type jsonArgs struct {
		url       string
		expiredAt time.Time
	}
	type mockArgs struct {
		wantInjectMock bool
		dbResult       driver.Result
		wantDBError    bool
	}
	tests := []struct {
		name               string
		jsonArgs           jsonArgs
		mockArgs           mockArgs
		expectedStatusCode int
	}{
		{
			"upload OK",
			jsonArgs{"http://example.com", validExpireTime},
			mockArgs{true, sqlmock.NewResult(1, 1), false},
			http.StatusOK,
		},
		{
			"internal db error",
			jsonArgs{"http://example.com", validExpireTime},
			mockArgs{true, nil, true},
			http.StatusInternalServerError,
		},
		{
			"invalid url",
			jsonArgs{"foobar", validExpireTime},
			mockArgs{false, nil, false},
			http.StatusBadRequest,
		},
		{
			"empty url",
			jsonArgs{"", validExpireTime},
			mockArgs{false, nil, false},
			http.StatusBadRequest,
		},
		{
			"upload an url with expired time",
			jsonArgs{"http://example.com", expiredTime},
			mockArgs{false, nil, false},
			http.StatusBadRequest,
		},
	}

	injectMock := func(mock sqlmock.Sqlmock, jsonArgs jsonArgs, result driver.Result, wantDBError bool) {
		// because ID generator is called in background, so turn off the in-order mode
		mock.MatchExpectationsInOrder(false)

		mock.ExpectBegin() // called by gorm
		exec := mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "urls" ("id","url","expired_at","created_at","updated_at","deleted_at") VALUES ($1,$2,$3,$4,$5,$6)`))
		if !wantDBError {
			// convert to and back
			expiredAtStr := jsonArgs.expiredAt.Format(expireAtLayout)
			expiredAt, _ := time.Parse(expireAtLayout, expiredAtStr)

			exec.
				WithArgs(anyValidID{}, jsonArgs.url, expiredAt, anyExpireTime{}, anyExpireTime{}, nil).
				WillReturnResult(result)
			mock.ExpectCommit() // called by gorm
		} else {
			exec.WillReturnError(errInternalDBError)
			mock.ExpectRollback() // called by gorm
		}
		// this sql will be called in background
		query := mock.ExpectQuery(regexp.QuoteMeta(`SELECT "id" FROM "urls" WHERE deleted_at IS NOT NULL OR expired_at < $1`))
		query.WithArgs(anyExpireTime{}).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqJSON := fmt.Sprintf(
				`{"url": "%s", "expireAt": "%s"}`,
				tt.jsonArgs.url, tt.jsonArgs.expiredAt.Format(expireAtLayout),
			)

			r := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(r)
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqJSON))

			gormDB, mock := getMockDB(t)
			if tt.mockArgs.wantInjectMock {
				injectMock(mock, tt.jsonArgs, tt.mockArgs.dbResult, tt.mockArgs.wantDBError)
			}

			u := UrlController{
				DB:             gormDB,
				Log:            logger,
				IDGenerator:    idgenerator.New(gormDB, logger),
				RedirectOrigin: redirectOrigin,
			}
			u.Upload(c)
			assert.Equal(t, tt.expectedStatusCode, r.Code)
			assert.NoError(t, mock.ExpectationsWereMet())

			if r.Code == http.StatusOK {
				var resp struct {
					ID       string `json:"id"`
					ShortUrl string `json:"shortUrl"`
				}
				err := json.Unmarshal(r.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.ID)
				assert.NotEmpty(t, resp.ShortUrl)
				assert.Equal(t, fmt.Sprintf("%s/%s", redirectOrigin, resp.ID), resp.ShortUrl)
			}
		})
	}
}

func TestUrlController_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name               string
		id                 string
		wantInjectMock     bool
		dbResult           driver.Result
		wantDBError        bool
		expectedStatusCode int
	}{
		{
			"delete ok",
			"okokok",
			true,
			sqlmock.NewResult(1, 1),
			false,
			http.StatusNoContent,
		},
		{
			"not found id",
			"noooid",
			true,
			sqlmock.NewResult(1, 0),
			false,
			http.StatusNotFound,
		},
		{
			"empty id",
			"",
			false,
			nil,
			false,
			http.StatusBadRequest,
		},
		{
			"internal db error",
			"okokok",
			true,
			nil,
			true,
			http.StatusInternalServerError,
		},
	}

	injectMock := func(mock sqlmock.Sqlmock, id string, result driver.Result, wantDBError bool) {
		mock.ExpectBegin() // called by gorm
		exec := mock.ExpectExec(regexp.QuoteMeta(`UPDATE "urls" SET "deleted_at"=$1 WHERE "urls"."id" = $2 AND "urls"."deleted_at" IS NULL`))
		if !wantDBError {
			exec.WithArgs(anyExpireTime{}, id).
				WillReturnResult(result)
			mock.ExpectCommit() // called by gorm
		} else {
			exec.WillReturnError(errInternalDBError)
			mock.ExpectRollback() // called by gorm
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(r)
			c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
			c.Params = []gin.Param{{Key: "url_id", Value: tt.id}}

			gormDB, mock := getMockDB(t)
			if tt.wantInjectMock {
				injectMock(mock, tt.id, tt.dbResult, tt.wantDBError)
			}

			u := UrlController{
				DB:             gormDB,
				Log:            logger,
				IDGenerator:    idgenerator.New(gormDB, logger),
				RedirectOrigin: "",
			}
			u.Delete(c)
			assert.Equal(t, tt.expectedStatusCode, r.Code)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUrlController_Redirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name               string
		id                 string
		wantInjectMock     bool
		dbErr              error
		expectedURL        string
		expectedStatusCode int
	}{
		{
			"redirect OK",
			"aaaaaa",
			true,
			nil,
			"https://example.com",
			http.StatusMovedPermanently,
		},
		{
			"empty id",
			"",
			false,
			nil,
			"",
			http.StatusBadRequest,
		},
		{
			"record not found",
			"nooURL",
			true,
			gorm.ErrRecordNotFound,
			"",
			http.StatusNotFound,
		},
		{
			"internal db error",
			"okokok",
			true,
			errInternalDBError,
			"",
			http.StatusInternalServerError,
		},
	}

	injectMock := func(mock sqlmock.Sqlmock, id, wantURL string, dbErr error) {
		exec := mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "urls" WHERE (id = $1 AND expired_at > $2) AND "urls"."deleted_at" IS NULL LIMIT 1`))
		if dbErr == nil {
			rows := sqlmock.NewRows([]string{"url"}).AddRow(wantURL)
			exec.WithArgs(id, anyExpireTime{}).
				WillReturnRows(rows)
		} else {
			exec.WillReturnError(dbErr)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(r)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Params = []gin.Param{{Key: "url_id", Value: tt.id}}
			gormDB, mock := getMockDB(t)
			if tt.wantInjectMock {
				injectMock(mock, tt.id, tt.expectedURL, tt.dbErr)
			}

			u := UrlController{
				DB:             gormDB,
				Log:            logger,
				IDGenerator:    idgenerator.New(gormDB, logger),
				RedirectOrigin: "",
			}
			u.Redirect(c)
			assert.Equal(t, tt.expectedStatusCode, r.Code)
			assert.Equal(t, tt.expectedURL, r.Header().Get("location"))
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
