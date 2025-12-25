package main

import (
    "os"

    "go.uber.org/zap"

    "github.com/lost-woods/random/src/rng"
    "github.com/lost-woods/random/src/server"
)

func main() {
    zapLogger, _ := zap.NewProduction()
    log := zapLogger.Sugar()
    defer func() { _ = zapLogger.Sync() }()

    // RNG (serial) init + initial health check
    srcRNG, health, err := rng.NewSerialRNGFromEnv()
    if err != nil {
        log.Fatal(err)
    }

    // Serialize access to the RNG stream across concurrent requests and health checks.
    r := rng.NewLockedReader(srcRNG)

    // Build + run server
    port := os.Getenv("PORT")
    if port == "" {
        port = "777"
    }
    s := server.New(port, r, health, log)
    s.RunOrDie()
}
