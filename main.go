package main

import (
	"fmt"
	"goshorturl/config"
	"goshorturl/logger"
	"goshorturl/repository"
	"goshorturl/server"
	"log"

	"go.uber.org/zap"
)

var (
	env       config.Env
	db        repository.Repository
	zaplogger *zap.Logger
)

func main() {
	var err error
	zaplogger, err = logger.Init()
	if err != nil {
		log.Fatalf("failed to initialize logger: %s", err)
	}

	env, err = config.Process()
	if err != nil {
		log.Fatalf("failed to process env: %s", err)
	}

	db, err = repository.InitPGRepo(env.DBPort, env.DBHost, env.DBUser, env.DBName, env.DBPassword)
	if err != nil {
		log.Fatalf("failed to connect db: %s", err)
	}

	r := server.NewRouter(db, zaplogger, env.RedirectOrigin)
	r.Run(fmt.Sprintf(":%d", env.AppPort))
}
