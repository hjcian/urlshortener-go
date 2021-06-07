package e2e

import (
	"goshorturl/config"
	"goshorturl/logger"
	"goshorturl/repository"
	"goshorturl/server"
	"log"
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
)

func Test_Server_Health(t *testing.T) {
	zaplogger, err := logger.Init()
	if err != nil {
		log.Fatalf("failed to initialize logger: %s", err)
	}

	env, err := config.Process()
	if err != nil {
		log.Fatalf("failed to process env: %s", err)
	}

	db, err := repository.InitPGRepo(env.DBPort, env.DBHost, env.DBUser, env.DBName, env.DBPassword)
	if err != nil {
		log.Fatalf("failed to connect db: %s", err)
	}

	engine := server.NewRouter(db, zaplogger, env.RedirectOrigin)

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

	// Assert response
	e.GET("/health").
		Expect().
		Status(http.StatusOK).JSON().Object().ValueEqual("status", "ok")
}
