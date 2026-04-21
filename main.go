package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/router"
)

func main() {
	// Load configuration from environment variables
	config.LoadConfig()

	// Initialize logger
	logger.SetupLogger()
	logger.SysLog(fmt.Sprintf("New API %s started", common.Version))

	// Set Gin mode based on environment
	// Default to release mode for better performance; set GIN_MODE=debug to override
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database connection
	err := model.InitDB()
	if err != nil {
		logger.FatalLog(fmt.Sprintf("Failed to initialize database: %s", err.Error()))
	}
	defer model.CloseDB()

	// Run database migrations
	err = model.MigrateDB()
	if err != nil {
		logger.FatalLog(fmt.Sprintf("Failed to migrate database: %s", err.Error()))
	}

	// Initialize cache (Redis or in-memory)
	// Note: Redis errors are non-fatal; the app falls back to in-memory cache gracefully
	err = common.InitRedisClient()
	if err != nil {
		logger.SysLog(fmt.Sprintf("Redis not available, using in-memory cache: %s", err.Error()))
	}

	// Initialize options from database
	model.InitOptionMap()

	// Start background tasks
	go model.SyncOptions(config.SyncFrequency)
	go controller.AutomaticDisableChannel()
	go controller.AutomaticEnableChannel()

	// Setup Gin router with middleware
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestId())
	middleware.SetUpLogger(engine)

	// Register all routes
	router.SetRouter(engine)

	// Determine port — prefer PORT env var, fall back to config value (default: 3000)
	port := os.Getenv("PORT")
	if port == "" {
		port = strconv.Itoa(config.Port)
	}

	logger.SysLog(fmt.Sprintf("Server listening on port %s", port))

	// Start HTTP server
	err = engine.Run(":" + port)
	if err != nil {
		logger.FatalLog(fmt.Sprintf("Failed to start server: %s", err.Error()))
	}
}
