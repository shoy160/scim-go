package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

// Config 全局配置结构体（映射config.yaml）
type Config struct {
	Mode     string `yaml:"mode"`
	Port     string `yaml:"port"`
	Token    string `yaml:"token"`
	LogLevel string `yaml:"log_level"`

	Storage struct {
		Driver              string `yaml:"driver"`
		RedisURI            string `yaml:"redis_uri"`
		MySQLDSN            string `yaml:"mysql_dsn"`
		PostgresDSN         string `yaml:"postgres_dsn"`
		AuthingHost         string `yaml:"authing_host"`
		AuthingUserPoolID   string `yaml:"authing_user_pool_id"`
		AuthingAccessKey    string `yaml:"authing_access_key"`
		AuthingAccessSecret string `yaml:"authing_access_secret"`
	} `yaml:"storage"`

	Pagination struct {
		DefaultCount  int  `yaml:"default_count"`
		MaxCount      int  `yaml:"max_count"`
		CursorSupport bool `yaml:"cursor_support"`
	} `yaml:"pagination"`

	SCIM struct {
		DefaultSchema string `yaml:"default_schema"`
		GroupSchema   string `yaml:"group_schema"`
		ErrorSchema   string `yaml:"error_schema"`
		ListSchema    string `yaml:"list_schema"`
	} `yaml:"scim"`

	Swagger struct {
		Enabled bool   `yaml:"enabled"`
		Path    string `yaml:"path"`
	} `yaml:"swagger"`
}

// ConfigManager 配置管理器
type ConfigManager struct {
	prefix string
}

// NewConfigManager 创建配置管理器
func NewConfigManager(prefix string) *ConfigManager {
	return &ConfigManager{
		prefix: strings.TrimSuffix(prefix, "_"),
	}
}

// GetString 获取字符串类型的环境变量
func (cm *ConfigManager) GetString(key string, defaultValue string) string {
	fullKey := cm.prefix + "_" + strings.ToUpper(key)
	if val := os.Getenv(fullKey); val != "" {
		log.Printf("环境变量覆盖: %s = %s", fullKey, val)
		return val
	}
	return defaultValue
}

// GetInt 获取整数类型的环境变量
func (cm *ConfigManager) GetInt(key string, defaultValue int) int {
	fullKey := cm.prefix + "_" + strings.ToUpper(key)
	if val := os.Getenv(fullKey); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			log.Printf("环境变量覆盖: %s = %d", fullKey, intVal)
			return intVal
		}
		log.Printf("环境变量解析错误: %s = %s (不是有效的整数)", fullKey, val)
	}
	return defaultValue
}

// GetBool 获取布尔类型的环境变量
func (cm *ConfigManager) GetBool(key string, defaultValue bool) bool {
	fullKey := cm.prefix + "_" + strings.ToUpper(key)
	if val := os.Getenv(fullKey); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			log.Printf("环境变量覆盖: %s = %t", fullKey, boolVal)
			return boolVal
		}
		log.Printf("环境变量解析错误: %s = %s (不是有效的布尔值)", fullKey, val)
	}
	return defaultValue
}

// GetStringWithOptions 获取带选项的字符串环境变量
func (cm *ConfigManager) GetStringWithOptions(key string, defaultValue string, options []string) string {
	val := cm.GetString(key, defaultValue)
	if len(options) > 0 {
		valid := false
		for _, opt := range options {
			if val == opt {
				valid = true
				break
			}
		}
		if !valid {
			log.Printf("环境变量值无效: %s = %s (有效选项: %v)", cm.prefix+"_"+strings.ToUpper(key), val, options)
			return defaultValue
		}
	}
	return val
}

// GetEnv 获取当前环境
func (cm *ConfigManager) GetEnv(defaultEnv string) string {
	env := cm.GetString("MODE", defaultEnv)
	validEnvs := []string{"debug", "test", "release"}
	valid := false
	for _, e := range validEnvs {
		if env == e {
			valid = true
			break
		}
	}
	if !valid {
		log.Printf("环境变量值无效: %s_MODE = %s (有效选项: %v)", cm.prefix, env, validEnvs)
		return defaultEnv
	}
	return env
}

// IsDevelopment 检查是否为开发环境
func (cm *ConfigManager) IsDebug() bool {
	return cm.GetEnv("debug") == "debug"
}

// IsTest 检查是否为测试环境
func (cm *ConfigManager) IsTest() bool {
	return cm.GetEnv("test") == "test"
}

// IsRelease 检查是否为发布环境
func (cm *ConfigManager) IsRelease() bool {
	return cm.GetEnv("release") == "release"
}

// GetNestedKey 获取嵌套键的环境变量名
func (cm *ConfigManager) GetNestedKey(prefix, key string) string {
	return fmt.Sprintf("%s_%s", strings.ToUpper(prefix), strings.ToUpper(key))
}

// loadConfig 加载配置文件并应用环境变量覆盖
func LoadConfig(path string) (Config, bool) {
	var globalCfg Config

	// 1. 加载配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read config file failed: %s\n", err)
	}
	if err := yaml.Unmarshal(data, &globalCfg); err != nil {
		log.Fatalf("unmarshal config failed: %s\n", err)
	}

	// 2. 应用环境变量覆盖
	cm := NewConfigManager("SCIM")

	// 基础配置
	globalCfg.Mode = cm.GetEnv(globalCfg.Mode)
	globalCfg.Port = cm.GetString("PORT", globalCfg.Port)
	globalCfg.Token = cm.GetString("TOKEN", globalCfg.Token)
	globalCfg.LogLevel = cm.GetString("LOG_LEVEL", globalCfg.LogLevel)

	// 存储配置
	globalCfg.Storage.Driver = cm.GetStringWithOptions("STORAGE_DRIVER", globalCfg.Storage.Driver, []string{"memory", "redis", "mysql", "postgres", "authing"})
	globalCfg.Storage.RedisURI = cm.GetString("STORAGE_REDIS_URI", globalCfg.Storage.RedisURI)
	globalCfg.Storage.MySQLDSN = cm.GetString("STORAGE_MYSQL_DSN", globalCfg.Storage.MySQLDSN)
	globalCfg.Storage.PostgresDSN = cm.GetString("STORAGE_POSTGRES_DSN", globalCfg.Storage.PostgresDSN)
	globalCfg.Storage.AuthingHost = cm.GetString("STORAGE_AUTHING_HOST", globalCfg.Storage.AuthingHost)
	globalCfg.Storage.AuthingUserPoolID = cm.GetString("STORAGE_AUTHING_USER_POOL_ID", globalCfg.Storage.AuthingUserPoolID)
	globalCfg.Storage.AuthingAccessKey = cm.GetString("STORAGE_AUTHING_ACCESS_KEY", globalCfg.Storage.AuthingAccessKey)
	globalCfg.Storage.AuthingAccessSecret = cm.GetString("STORAGE_AUTHING_ACCESS_SECRET", globalCfg.Storage.AuthingAccessSecret)

	// 分页配置
	globalCfg.Pagination.DefaultCount = cm.GetInt("PAGINATION_DEFAULT_COUNT", globalCfg.Pagination.DefaultCount)
	globalCfg.Pagination.MaxCount = cm.GetInt("PAGINATION_MAX_COUNT", globalCfg.Pagination.MaxCount)
	globalCfg.Pagination.CursorSupport = cm.GetBool("PAGINATION_CURSOR_SUPPORT", globalCfg.Pagination.CursorSupport)

	// SCIM 配置
	globalCfg.SCIM.DefaultSchema = cm.GetString("SCIM_DEFAULT_SCHEMA", globalCfg.SCIM.DefaultSchema)
	globalCfg.SCIM.GroupSchema = cm.GetString("SCIM_GROUP_SCHEMA", globalCfg.SCIM.GroupSchema)
	globalCfg.SCIM.ErrorSchema = cm.GetString("SCIM_ERROR_SCHEMA", globalCfg.SCIM.ErrorSchema)
	globalCfg.SCIM.ListSchema = cm.GetString("SCIM_LIST_SCHEMA", globalCfg.SCIM.ListSchema)

	// Swagger 配置
	globalCfg.Swagger.Enabled = cm.GetBool("SWAGGER_ENABLED", globalCfg.Swagger.Enabled)
	globalCfg.Swagger.Path = cm.GetString("SWAGGER_PATH", globalCfg.Swagger.Path)
	// 测试模式：打印配置并退出
	if os.Getenv("TEST_ENV") == "true" {
		log.Println("=== 测试配置管理器 ===")
		log.Printf("Mode: %s\n", globalCfg.Mode)
		log.Printf("Port: %s\n", globalCfg.Port)
		log.Printf("Token: %s\n", globalCfg.Token)
		log.Printf("LogLevel: %s\n", globalCfg.LogLevel)
		log.Printf("Storage Driver: %s\n", globalCfg.Storage.Driver)
		log.Printf("Storage Redis URI: %s\n", globalCfg.Storage.RedisURI)
		log.Printf("Storage MySQL DSN: %s\n", globalCfg.Storage.MySQLDSN)
		log.Printf("Storage Postgres DSN: %s\n", globalCfg.Storage.PostgresDSN)
		log.Printf("Storage Authing Host: %s\n", globalCfg.Storage.AuthingHost)
		log.Printf("Storage Authing User Pool ID: %s\n", globalCfg.Storage.AuthingUserPoolID)
		log.Printf("Storage Authing Access Key: %s\n", globalCfg.Storage.AuthingAccessKey)
		log.Printf("Storage Authing Access Secret: %s\n", globalCfg.Storage.AuthingAccessSecret)
		log.Printf("Pagination Default Count: %d\n", globalCfg.Pagination.DefaultCount)
		log.Printf("Pagination Max Count: %d\n", globalCfg.Pagination.MaxCount)
		log.Printf("Pagination Cursor Support: %t\n", globalCfg.Pagination.CursorSupport)
		log.Printf("SCIM Default Schema: %s\n", globalCfg.SCIM.DefaultSchema)
		log.Printf("SCIM Group Schema: %s\n", globalCfg.SCIM.GroupSchema)
		log.Printf("SCIM Error Schema: %s\n", globalCfg.SCIM.ErrorSchema)
		log.Printf("SCIM List Schema: %s\n", globalCfg.SCIM.ListSchema)
		log.Println("=== 测试完成 ===")
		return globalCfg, true
	}
	return globalCfg, false
}
