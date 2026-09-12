package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Kilat-Pet-Delivery/lib-common/auth"
	"github.com/Kilat-Pet-Delivery/lib-common/database"
	"github.com/Kilat-Pet-Delivery/lib-common/health"
	"github.com/Kilat-Pet-Delivery/lib-common/kafka"
	"github.com/Kilat-Pet-Delivery/lib-common/logger"
	"github.com/Kilat-Pet-Delivery/lib-common/middleware"
	"github.com/Kilat-Pet-Delivery/service-runner/internal/application"
	svcConfig "github.com/Kilat-Pet-Delivery/service-runner/internal/config"
	runnerDomain "github.com/Kilat-Pet-Delivery/service-runner/internal/domain/runner"
	"github.com/Kilat-Pet-Delivery/service-runner/internal/handler"
	"github.com/Kilat-Pet-Delivery/service-runner/internal/repository"
)

func main() {
	// Load configuration.
	cfg, err := svcConfig.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger.
	zapLogger, err := logger.NewNamed(cfg.AppEnv, "service-runner")
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer func() { _ = zapLogger.Sync() }()

	// Connect to database.
	dbConfig := database.PostgresConfig{
		Host:     cfg.DBConfig.Host,
		Port:     cfg.DBConfig.Port,
		User:     cfg.DBConfig.User,
		Password: cfg.DBConfig.Password,
		DBName:   cfg.DBConfig.DBName,
		SSLMode:  cfg.DBConfig.SSLMode,
	}
	db, err := database.Connect(dbConfig, zapLogger)
	if err != nil {
		zapLogger.Fatal("failed to connect to database", zap.Error(err))
	}

	// Run database migrations.
	//
	// KPD-59: this used to AutoMigrate in development and run the SQL migrations
	// everywhere else. pet_shops had no SQL migration, so it existed only in
	// development. Now that 003_create_pet_shops covers it, every model in this
	// service has a SQL migration -- so there is one path for all environments and
	// the two can no longer drift apart. Development still gets its schema
	// automatically, because the server applies the migrations at startup.
	dbURL := dbConfig.DatabaseURL()
	if err := database.RunMigrations(dbURL, "migrations", zapLogger); err != nil {
		zapLogger.Fatal("failed to run migrations", zap.Error(err))
	}

	// Sample pet shops, for development only -- the directory is empty otherwise.
	if cfg.AppEnv == "development" {
		repository.SeedPetShops(db, zapLogger)
	}

	// Initialize JWT manager.
	accessExpiry, _ := time.ParseDuration(cfg.JWTConfig.AccessExpiry)
	if accessExpiry == 0 {
		accessExpiry = 15 * time.Minute
	}
	refreshExpiry, _ := time.ParseDuration(cfg.JWTConfig.RefreshExpiry)
	if refreshExpiry == 0 {
		refreshExpiry = 7 * 24 * time.Hour
	}
	jwtManager := auth.NewJWTManager(cfg.JWTConfig.Secret, accessExpiry, refreshExpiry)

	// Initialize Kafka producer.
	producer := kafka.NewProducer(cfg.KafkaConfig.Brokers, zapLogger)
	defer func() { _ = producer.Close() }()

	// Initialize repositories.
	runnerRepo := repository.NewGormRunnerRepository(db)
	crateSpecRepo := repository.NewGormCrateSpecRepository(db)

	// Initialize quality policy.
	qualityPolicy := runnerDomain.NewDefaultQualityPolicy()

	// Initialize application service.
	runnerService := application.NewRunnerService(
		runnerRepo,
		crateSpecRepo,
		qualityPolicy,
		producer,
		zapLogger,
	)

	// Initialize pet shop service and handler.
	petShopRepo := repository.NewGormPetShopRepository(db)
	petShopService := application.NewPetShopService(petShopRepo, zapLogger)
	petShopHandler := handler.NewPetShopHandler(petShopService)

	// Initialize handler.
	runnerHandler := handler.NewRunnerHandler(runnerService)

	// Set up Gin router.
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Apply global middleware.
	router.Use(middleware.RecoveryMiddleware(zapLogger))
	router.Use(middleware.LoggerMiddleware(zapLogger))
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.SecurityHeadersMiddleware())

	// Register health check routes.
	healthHandler := health.NewHandler(db, "service-runner")
	healthHandler.RegisterRoutes(router)

	// Register API routes.
	apiV1 := router.Group("/api/v1")
	runnerHandler.RegisterRoutes(apiV1, jwtManager)
	petShopHandler.RegisterRoutes(apiV1, jwtManager)

	// Create HTTP server.
	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine.
	go func() {
		zapLogger.Info("starting service-runner", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("shutting down service-runner...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Fatal("server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("service-runner stopped gracefully")
}
