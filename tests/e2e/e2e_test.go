package e2e

import (
	"goshorturl/config"
	"goshorturl/idgenerator"
	"goshorturl/logger"
	"goshorturl/repository"
	"goshorturl/server"
	"time"

	"log"
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
)

const expireAtLayout = "2006-01-02T15:04:05Z"

func Test_Server_Health(t *testing.T) {
	zaplogger, err := logger.New()
	if err != nil {
		log.Fatalf("failed to initialize logger: %s", err)
	}

	env, err := config.Process()
	if err != nil {
		log.Fatalf("failed to process env: %s", err)
	}

	db, err := repository.NewPG(env.DBPort, env.DBHost, env.DBUser, env.DBName, env.DBPassword)
	if err != nil {
		log.Fatalf("failed to connect db: %s", err)
	}
	idGenerator := idgenerator.New(db, zaplogger)

	engine := server.NewRouter(db, idGenerator, zaplogger, env.RedirectOrigin)

	e := httpexpect.WithConfig(httpexpect.Config{
		Client: &http.Client{
			Transport: httpexpect.NewBinder(engine),
			Jar:       httpexpect.NewJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		},
	})

	t.Run("health check", func(t *testing.T) {
		e.GET("/health").
			Expect().
			Status(http.StatusOK).JSON().Object().ValueEqual("status", "ok")
	})

	t.Run("1.upload(ok)=>2.redirect(ok)=>3.delete(ok)=>4.redirect(not found)", func(t *testing.T) {
		uploadedUrl := "http://example.com"

		req := map[string]interface{}{
			"url":      uploadedUrl,
			"expireAt": time.Now().Add(24 * time.Hour).Format(expireAtLayout),
		}
		// 1.
		obj := e.POST("/api/v1/urls").WithJSON(req).
			Expect().
			Status(http.StatusOK).
			JSON().Object()
		obj.Keys().ContainsOnly("id", "shortUrl")

		// 2.
		id := obj.Value("id").Raw()
		e.GET("/{id}", id).
			WithRedirectPolicy(httpexpect.DontFollowRedirects). // hint: https://github.com/gavv/httpexpect/blob/c8d94c7cd00324483558c1877cdcd6350138335e/e2e_redirect_test.go#L61
			Expect().
			StatusRange(httpexpect.Status3xx).
			Header("location").Equal(uploadedUrl)

		// 3.
		e.DELETE("/api/v1/urls/{id}", id).
			Expect().
			Status(http.StatusNoContent)

		// 4.
		e.GET("/{id}", id).
			Expect().
			StatusRange(http.StatusNotFound)
	})

	t.Run("1.upload A(ok, get idX)=>2.upload A again(ok, get idY)=>3.redirect idX to A(ok)=>4.redirect idY to A(ok)", func(t *testing.T) {
		uploadedUrl := "http://example.com"

		req := map[string]interface{}{
			"url":      uploadedUrl,
			"expireAt": time.Now().Add(24 * time.Hour).Format(expireAtLayout),
		}
		// 1.
		obj := e.POST("/api/v1/urls").WithJSON(req).
			Expect().
			Status(http.StatusOK).
			JSON().Object()
		obj.Keys().ContainsOnly("id", "shortUrl")
		idX := obj.Value("id").Raw()

		// 2.
		obj = e.POST("/api/v1/urls").WithJSON(req).
			Expect().
			Status(http.StatusOK).
			JSON().Object()
		obj.Keys().ContainsOnly("id", "shortUrl")
		idY := obj.Value("id").Raw()

		assert.NotEqual(t, idX, idY, "two di should different")

		// 3.
		e.GET("/{id}", idX).
			WithRedirectPolicy(httpexpect.DontFollowRedirects).
			Expect().
			StatusRange(httpexpect.Status3xx).
			Header("location").Equal(uploadedUrl)

		// 4.
		e.GET("/{id}", idY).
			WithRedirectPolicy(httpexpect.DontFollowRedirects).
			Expect().
			StatusRange(httpexpect.Status3xx).
			Header("location").Equal(uploadedUrl)
	})
}
