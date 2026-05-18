package main

import (
	"context"
	"feedsystem_ai_go/internal/config"
	"feedsystem_ai_go/internal/db"
	apphttp "feedsystem_ai_go/internal/http"
	rabbitmq "feedsystem_ai_go/internal/middleware/rabbitmq"
	rediscache "feedsystem_ai_go/internal/middleware/redis"
	"feedsystem_ai_go/internal/observability"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	goredis "github.com/redis/go-redis/v9"
)

func main() {
	// 加载 .env（本地开发）
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found; continuing")
	}

	// 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	log.Printf("Loading config from %s", configPath)
	cfg, usedDefault, err := config.LoadLocalDev(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if usedDefault {
		log.Printf("Config File %s not found, using default local config", configPath)
	} else {
		log.Printf("Config loaded from file: %s", configPath)
	}

	// 连接数据库
	//log.Printf("Database config: %v", cfg.Database)
	sqlDB, err := db.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(sqlDB); err != nil {
		log.Fatalf("Failed to auto migrate database: %v", err)
	}
	defer db.CloseDB(sqlDB)

	// 连接 Redis (可选，用于缓存)
	cache, err := rediscache.NewFromEnv(&cfg.Redis)
	if err != nil {
		log.Printf("Redis config error (cache disabled): %v", err)
		cache = nil
	} else {
		pingCtx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		if err := cache.Ping(pingCtx); err != nil {
			log.Printf("Redis not available (cache disabled): %v", err)
			_ = cache.Close()
			cache = nil
		} else {
			defer cache.Close()
			log.Printf("Redis connected (cache enabled)")
		}
	}

	// 连接 RabbitMQ (可选，用于消息队列)
	rmq, err := rabbitmq.NewRabbitMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Printf("RabbitMQ config error (disabled): %v", err)
		rmq = nil
	} else {
		defer rmq.Close()
		log.Printf("RabbitMQ connected")
	}
	// Pprof
	pprofServer, err := observability.NewPprofServer(
		"API",
		cfg.ObservabilityConfig.Pprof.Enabled,
		cfg.ObservabilityConfig.Pprof.ApiAddr,
	)
	if err != nil {
		log.Printf("Failed to start API pprof server: %v", err)
	}
	if pprofServer != nil {
		defer pprofServer.Close()
	}

	// 创建独立的 go-redis 客户端 (用于 AI/媒体模块的令牌桶、分布式锁等)
	var rdb *goredis.Client
	if cfg.Redis.Host != "" {
		rdb = goredis.NewClient(&goredis.Options{
			Addr:     cfg.Redis.Host + ":" + strconv.Itoa(cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		pingCtx2, cancel2 := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel2()
		if err := rdb.Ping(pingCtx2).Err(); err != nil {
			log.Printf("Redis go-client not available (AI/Media rate-limit disabled): %v", err)
			rdb.Close()
			rdb = nil
		} else {
			defer rdb.Close()
			log.Printf("Redis go-client connected (AI/Media rate-limit enabled)")
		}
	}

	// 设置路由
	r := apphttp.SetRouter(sqlDB, cache, rdb, rmq, cfg)
	log.Printf("Server is running on port %d", cfg.Server.Port)
	if err := r.Run(":" + strconv.Itoa(cfg.Server.Port)); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
