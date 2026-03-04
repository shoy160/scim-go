// @title SCIM 2.0 API
// @version 1.0.0
// @description SCIM (System for Cross-domain Identity Management) 2.0 实现，提供用户和组管理功能
// @termsOfService https://example.com/terms

// @contact.name shoy160
// @contact.email shoy160@qq.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /scim/v2

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 请输入 "Bearer {token}" 格式的认证令牌

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"scim-go/api"
	"scim-go/internal/config"
	"scim-go/store"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 全局配置实例
var globalCfg config.Config

// 全局路由小写化中间件：强制所有路径转小写，100% 不区分大小写
func LowerCasePath() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 把整个 URL 路径全部转小写
		c.Request.URL.Path = strings.ToLower(c.Request.URL.Path)
		c.Next()
	}
}

// initStorage 初始化存储
func initStorage(cfg config.Config) store.Store {
	switch cfg.Storage.Driver {
	case "redis":
		s := store.NewRedis(cfg.Storage.RedisURI)
		log.Println("storage driver: redis")
		return s
	case "mysql":
		s := initMySQL(cfg)
		log.Println("storage driver: mysql")
		return s
	case "postgres":
		s := initPostgres(cfg)
		log.Println("storage driver: postgres")
		return s
	case "authing":
		s := store.NewAuthingStore(
			cfg.Storage.AuthingHost,
			cfg.Storage.AuthingUserPoolID,
			cfg.Storage.AuthingAccessKey,
			cfg.Storage.AuthingAccessSecret,
		)
		log.Println("storage driver: authing")
		return s
	default:
		s := store.NewMemory()
		log.Println("storage driver: memory (default)")
		return s
	}
}

// initMySQL 初始化MySQL连接
func initMySQL(cfg config.Config) store.Store {
	// 配置Gorm日志
	gormLogger := logger.Default
	if cfg.Mode == gin.ReleaseMode {
		gormLogger = gormLogger.LogMode(logger.Error)
	}
	db, err := gorm.Open(mysql.Open(cfg.Storage.MySQLDSN), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("mysql connect failed: %s\n", err)
	}
	return store.NewDB(db, nil)
}

// initPostgres 初始化PostgreSQL连接
func initPostgres(cfg config.Config) store.Store {
	gormLogger := logger.Default
	if cfg.Mode == gin.ReleaseMode {
		gormLogger = gormLogger.LogMode(logger.Error)
	}
	db, err := gorm.Open(postgres.Open(cfg.Storage.PostgresDSN), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("postgres connect failed: %s\n", err)
	}
	return store.NewDB(db, nil)
}

func main() {
	// 1. 加载配置文件
	var configTest bool
	globalCfg, configTest = config.LoadConfig("./config.yaml")
	if configTest {
		return
	}

	// 2. 设置Gin运行模式
	gin.SetMode(globalCfg.Mode)
	r := gin.New()
	// 加入基础中间件：日志+恢复panic
	r.Use(LowerCasePath(), gin.Logger(), gin.Recovery())

	// 3. 初始化存储
	s := initStorage(globalCfg)

	// 4. 注册SCIM接口
	scimCfg := &api.ScimConfig{
		DefaultSchema: globalCfg.SCIM.DefaultSchema,
		GroupSchema:   globalCfg.SCIM.GroupSchema,
		ErrorSchema:   globalCfg.SCIM.ErrorSchema,
		ListSchema:    globalCfg.SCIM.ListSchema,
		DefaultCount:  globalCfg.Pagination.DefaultCount,
		MaxCount:      globalCfg.Pagination.MaxCount,
	}
	api.RegisterRoutes(r, s, scimCfg, globalCfg.Token, globalCfg.Swagger.Enabled, globalCfg.Swagger.Path)

	// 5. 启动HTTP服务
	srv := &http.Server{
		Addr:    ":" + globalCfg.Port,
		Handler: r,
	}

	// 启动服务
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen failed: %s\n", err)
		}
	}()
	log.Printf("scim server start on :%s (mode: %s)\n", globalCfg.Port, globalCfg.Mode)

	// 6. 优雅关闭服务（监听信号）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit
	log.Println("server is shutting down...")

	// 关闭上下文（5秒超时）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server shutdown failed: ", err)
	}
	log.Println("server shutdown successfully")
}
