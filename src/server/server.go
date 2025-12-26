package server

import (
	"io"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/lost-woods/random/src/api"
	"github.com/lost-woods/random/src/rng"
)

type Server struct {
	port   string
	router *gin.Engine
}

func New(port string, r io.Reader, h *rng.Health, log *zap.SugaredLogger) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Background health monitoring (best-effort)
	// Interval is configurable via RNG_HEALTH_INTERVAL (default 10000ms).
	interval := 10_000 * time.Millisecond
	if msStr := os.Getenv("RNG_HEALTH_INTERVAL"); msStr != "" {
		if ms, err := strconv.Atoi(msStr); err == nil && ms > 0 {
			interval = time.Duration(ms) * time.Millisecond
		}
	}
	go rng.PeriodicHealthCheck(r, h, interval)

	router.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET"},
		AllowHeaders:     []string{"X-API-KEY", "Accept"},
		AllowAllOrigins:  true,
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))
	router.Use(api.CheckHeader("X-API-KEY", api.APIKeyFromEnv()))

	handlers := api.NewHandlers(r, h, log)
	router.GET("/", handlers.RandomNumber)
	router.GET("/bytes", handlers.RandomBytes)
	router.GET("/cards", handlers.RandomCards)
	router.GET("/strings", handlers.RandomStrings)
	router.GET("/percent", handlers.RandomPercent)
	router.GET("/health", handlers.Health)

	return &Server{port: port, router: router}
}

func (s *Server) RunOrDie() {
	if err := s.router.Run(":" + s.port); err != nil {
		panic(err)
	}
}
