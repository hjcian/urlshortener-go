package main

import (
	"context"
	"fmt"
	"goshorturl/cache"
	"goshorturl/config"
	"goshorturl/idgenerator"
	"goshorturl/logger"
	"goshorturl/repository"
	"goshorturl/server"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	env       config.Env
	db        repository.Repository
	zaplogger *zap.Logger
)

func main() {
	var err error
	zaplogger, err = logger.New()
	if err != nil {
		log.Fatalf("failed to initialize logger: %s", err)
	}

	env, err = config.Process()
	if err != nil {
		log.Fatalf("failed to process env: %s", err)
	}

	db, err = repository.NewPG(env.DBPort, env.DBHost, env.DBUser, env.DBName, env.DBPassword)
	if err != nil {
		log.Fatalf("failed to connect db: %s", err)
	}

	cache := cache.New(db, zaplogger, cache.UseRedis(env.CacheHost, env.CachePort))
	idGenerator := idgenerator.New(cache, zaplogger)

	r := server.NewRouter(cache, idGenerator, zaplogger, env.RedirectOrigin)
	run(r, fmt.Sprintf(":%d", env.AppPort))
}

func run(r *gin.Engine, addr string) {
	// Graceful stop: https://gin-gonic.com/docs/examples/graceful-restart-or-stop/
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	// catching ctx.Done(). timeout of 3 seconds.
	select {
	case <-ctx.Done():
		log.Println("timeout of 3 seconds.")
	}
	log.Println("Server exiting")
}
