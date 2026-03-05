package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/joho/godotenv"
)

// ConfigSource 配置来源类型
type ConfigSource int

const (
	SourceDefault ConfigSource = iota
	SourceEnvFile
	SourceConfigFile
	SourceEnvVar
	SourceCLI
)

func (cs ConfigSource) String() string {
	switch cs {
	case SourceDefault:
		return "default"
	case SourceEnvFile:
		return ".env file"
	case SourceConfigFile:
		return "config file"
	case SourceEnvVar:
		return "environment variable"
	case SourceCLI:
		return "command line"
	default:
		return "unknown"
	}
}

// ConfigField 配置字段信息
type ConfigField struct {
	Name         string
	Value        interface{}
	Source       ConfigSource
	IsSensitive  bool
	IsEncrypted  bool
	Description  string
	DefaultValue interface{}
	ValidOptions []string
	Validator    func(interface{}) error
}

// ConfigRegistry 配置注册表
type ConfigRegistry struct {
	fields map[string]*ConfigField
}

// NewConfigRegistry 创建配置注册表
func NewConfigRegistry() *ConfigRegistry {
	return &ConfigRegistry{
		fields: make(map[string]*ConfigField),
	}
}

// Register 注册配置字段
func (cr *ConfigRegistry) Register(field *ConfigField) {
	cr.fields[field.Name] = field
}

// Get 获取配置字段
func (cr *ConfigRegistry) Get(name string) (*ConfigField, bool) {
	field, ok := cr.fields[name]
	return field, ok
}

// GetAll 获取所有配置字段
func (cr *ConfigRegistry) GetAll() map[string]*ConfigField {
	return cr.fields
}

// CLIArgs 命令行参数结构
type CLIArgs struct {
	ConfigPath     *string
	Port           *string
	Token          *string
	StorageDriver  *string
	Mode           *string
	LogLevel       *string
	RedisURI       *string
	MySQLDSN       *string
	PostgresDSN    *string
	SwaggerEnabled *bool
	ShowHelp       *bool
	ShowVersion    *bool
}

// GlobalCLIArgs 全局命令行参数实例
var GlobalCLIArgs CLIArgs

// InitCLIArgs 初始化命令行参数
func InitCLIArgs() {
	GlobalCLIArgs.ConfigPath = flag.String("config", "", "配置文件路径 (支持 .yaml, .json, .env)")
	GlobalCLIArgs.Port = flag.String("port", "", "服务端口号")
	GlobalCLIArgs.Token = flag.String("token", "", "身份验证令牌 (支持加密格式: enc:base64encoded)")
	GlobalCLIArgs.StorageDriver = flag.String("storage", "", "数据存储方式 (memory/redis/mysql/postgres/authing)")
	GlobalCLIArgs.Mode = flag.String("mode", "", "运行模式 (debug/test/release)")
	GlobalCLIArgs.LogLevel = flag.String("log-level", "", "日志级别 (debug/info/warn/error)")
	GlobalCLIArgs.RedisURI = flag.String("redis-uri", "", "Redis连接URI")
	GlobalCLIArgs.MySQLDSN = flag.String("mysql-dsn", "", "MySQL DSN连接字符串")
	GlobalCLIArgs.PostgresDSN = flag.String("postgres-dsn", "", "PostgreSQL DSN连接字符串")
	GlobalCLIArgs.SwaggerEnabled = flag.Bool("swagger", false, "启用Swagger文档")
	GlobalCLIArgs.ShowHelp = flag.Bool("help", false, "显示帮助信息")
	GlobalCLIArgs.ShowVersion = flag.Bool("version", false, "显示版本信息")

	flag.Parse()
}

// LoadConfigWithPriority 按优先级加载配置
// 优先级: 命令行参数 > 环境变量 > 配置文件 > .env文件 > 默认值
func LoadConfigWithPriority(configPath string) (*Config, *ConfigRegistry, error) {
	registry := NewConfigRegistry()
	config := &Config{}

	// 1. 设置默认值
	setDefaults(config, registry)

	// 2. 加载 .env 文件（如果存在）
	if err := loadEnvFile(config, registry); err != nil {
		log.Printf("警告: 加载 .env 文件失败: %v", err)
	}

	// 3. 加载配置文件（如果指定）
	if configPath != "" {
		if err := loadConfigFile(configPath, config, registry); err != nil {
			return nil, nil, fmt.Errorf("加载配置文件失败: %w", err)
		}
	}

	// 4. 加载环境变量
	loadFromEnvironment(config, registry)

	// 5. 加载命令行参数（最高优先级）
	loadFromCLI(config, registry)

	// 6. 验证配置
	if err := validateConfig(config, registry); err != nil {
		return nil, nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 7. 解密敏感配置
	if err := decryptSensitiveFields(config, registry); err != nil {
		return nil, nil, fmt.Errorf("解密敏感配置失败: %w", err)
	}

	return config, registry, nil
}

// setDefaults 设置默认配置值
func setDefaults(cfg *Config, registry *ConfigRegistry) {
	defaults := map[string]interface{}{
		"mode":                       "debug",
		"port":                       "8080",
		"log_level":                  "info",
		"storage.driver":             "memory",
		"pagination.default_count":   20,
		"pagination.max_count":       100,
		"pagination.cursor_support":  true,
		"scim.default_schema":        "urn:ietf:params:scim:schemas:core:2.0:User",
		"scim.group_schema":          "urn:ietf:params:scim:schemas:core:2.0:Group",
		"scim.error_schema":          "urn:ietf:params:scim:api:messages:2.0:Error",
		"scim.list_schema":           "urn:ietf:params:scim:api:messages:2.0:ListResponse",
		"swagger.enabled":            true,
		"swagger.path":               "/swagger/*any",
	}

	cfg.Mode = "debug"
	cfg.Port = "8080"
	cfg.LogLevel = "info"
	cfg.Storage.Driver = "memory"
	cfg.Pagination.DefaultCount = 20
	cfg.Pagination.MaxCount = 100
	cfg.Pagination.CursorSupport = true
	cfg.SCIM.DefaultSchema = "urn:ietf:params:scim:schemas:core:2.0:User"
	cfg.SCIM.GroupSchema = "urn:ietf:params:scim:schemas:core:2.0:Group"
	cfg.SCIM.ErrorSchema = "urn:ietf:params:scim:api:messages:2.0:Error"
	cfg.SCIM.ListSchema = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	cfg.Swagger.Enabled = true
	cfg.Swagger.Path = "/swagger/*any"

	for name, value := range defaults {
		registry.Register(&ConfigField{
			Name:         name,
			Value:        value,
			Source:       SourceDefault,
			DefaultValue: value,
		})
	}
}

// loadEnvFile 从 .env 文件加载配置
func loadEnvFile(cfg *Config, registry *ConfigRegistry) error {
	envPaths := []string{".env", ".env.local", ".env." + cfg.Mode}

	for _, path := range envPaths {
		if _, err := os.Stat(path); err == nil {
			if err := godotenv.Load(path); err != nil {
				return fmt.Errorf("加载 %s 失败: %w", path, err)
			}
			log.Printf("已加载环境变量文件: %s", path)
		}
	}

	return nil
}

// loadConfigFile 从配置文件加载配置
func loadConfigFile(path string, cfg *Config, registry *ConfigRegistry) error {
	ext := strings.ToLower(filepath.Ext(path))

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("解析 YAML 失败: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("解析 JSON 失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的配置文件格式: %s", ext)
	}

	// 更新注册表中的配置来源
	updateRegistryFromConfig(cfg, registry, SourceConfigFile)

	log.Printf("已加载配置文件: %s", path)
	return nil
}

// loadFromEnvironment 从环境变量加载配置
func loadFromEnvironment(cfg *Config, registry *ConfigRegistry) {
	cm := NewConfigManager("SCIM")

	// 基础配置
	if val := cm.GetString("MODE", cfg.Mode); val != cfg.Mode {
		cfg.Mode = val
		registry.Register(&ConfigField{Name: "mode", Value: val, Source: SourceEnvVar})
	}
	if val := cm.GetString("PORT", cfg.Port); val != cfg.Port {
		cfg.Port = val
		registry.Register(&ConfigField{Name: "port", Value: val, Source: SourceEnvVar})
	}
	if val := cm.GetString("TOKEN", cfg.Token); val != cfg.Token {
		cfg.Token = val
		registry.Register(&ConfigField{Name: "token", Value: val, Source: SourceEnvVar, IsSensitive: true})
	}
	if val := cm.GetString("LOG_LEVEL", cfg.LogLevel); val != cfg.LogLevel {
		cfg.LogLevel = val
		registry.Register(&ConfigField{Name: "log_level", Value: val, Source: SourceEnvVar})
	}

	// 存储配置
	if val := cm.GetStringWithOptions("STORAGE_DRIVER", cfg.Storage.Driver, []string{"memory", "redis", "mysql", "postgres", "authing"}); val != cfg.Storage.Driver {
		cfg.Storage.Driver = val
		registry.Register(&ConfigField{Name: "storage.driver", Value: val, Source: SourceEnvVar, ValidOptions: []string{"memory", "redis", "mysql", "postgres", "authing"}})
	}
	if val := cm.GetString("STORAGE_REDIS_URI", cfg.Storage.RedisURI); val != cfg.Storage.RedisURI {
		cfg.Storage.RedisURI = val
		registry.Register(&ConfigField{Name: "storage.redis_uri", Value: val, Source: SourceEnvVar, IsSensitive: true})
	}
	if val := cm.GetString("STORAGE_MYSQL_DSN", cfg.Storage.MySQLDSN); val != cfg.Storage.MySQLDSN {
		cfg.Storage.MySQLDSN = val
		registry.Register(&ConfigField{Name: "storage.mysql_dsn", Value: val, Source: SourceEnvVar, IsSensitive: true})
	}
	if val := cm.GetString("STORAGE_POSTGRES_DSN", cfg.Storage.PostgresDSN); val != cfg.Storage.PostgresDSN {
		cfg.Storage.PostgresDSN = val
		registry.Register(&ConfigField{Name: "storage.postgres_dsn", Value: val, Source: SourceEnvVar, IsSensitive: true})
	}

	// 分页配置
	if val := cm.GetInt("PAGINATION_DEFAULT_COUNT", cfg.Pagination.DefaultCount); val != cfg.Pagination.DefaultCount {
		cfg.Pagination.DefaultCount = val
		registry.Register(&ConfigField{Name: "pagination.default_count", Value: val, Source: SourceEnvVar})
	}
	if val := cm.GetInt("PAGINATION_MAX_COUNT", cfg.Pagination.MaxCount); val != cfg.Pagination.MaxCount {
		cfg.Pagination.MaxCount = val
		registry.Register(&ConfigField{Name: "pagination.max_count", Value: val, Source: SourceEnvVar})
	}

	// Swagger 配置
	if val := cm.GetBool("SWAGGER_ENABLED", cfg.Swagger.Enabled); val != cfg.Swagger.Enabled {
		cfg.Swagger.Enabled = val
		registry.Register(&ConfigField{Name: "swagger.enabled", Value: val, Source: SourceEnvVar})
	}
}

// loadFromCLI 从命令行参数加载配置
func loadFromCLI(cfg *Config, registry *ConfigRegistry) {
	if GlobalCLIArgs.Port != nil && *GlobalCLIArgs.Port != "" {
		cfg.Port = *GlobalCLIArgs.Port
		registry.Register(&ConfigField{Name: "port", Value: *GlobalCLIArgs.Port, Source: SourceCLI})
	}

	if GlobalCLIArgs.Token != nil && *GlobalCLIArgs.Token != "" {
		cfg.Token = *GlobalCLIArgs.Token
		registry.Register(&ConfigField{Name: "token", Value: *GlobalCLIArgs.Token, Source: SourceCLI, IsSensitive: true, IsEncrypted: strings.HasPrefix(*GlobalCLIArgs.Token, "enc:")})
	}

	if GlobalCLIArgs.StorageDriver != nil && *GlobalCLIArgs.StorageDriver != "" {
		cfg.Storage.Driver = *GlobalCLIArgs.StorageDriver
		registry.Register(&ConfigField{Name: "storage.driver", Value: *GlobalCLIArgs.StorageDriver, Source: SourceCLI, ValidOptions: []string{"memory", "redis", "mysql", "postgres", "authing"}})
	}

	if GlobalCLIArgs.Mode != nil && *GlobalCLIArgs.Mode != "" {
		cfg.Mode = *GlobalCLIArgs.Mode
		registry.Register(&ConfigField{Name: "mode", Value: *GlobalCLIArgs.Mode, Source: SourceCLI})
	}

	if GlobalCLIArgs.LogLevel != nil && *GlobalCLIArgs.LogLevel != "" {
		cfg.LogLevel = *GlobalCLIArgs.LogLevel
		registry.Register(&ConfigField{Name: "log_level", Value: *GlobalCLIArgs.LogLevel, Source: SourceCLI})
	}

	if GlobalCLIArgs.RedisURI != nil && *GlobalCLIArgs.RedisURI != "" {
		cfg.Storage.RedisURI = *GlobalCLIArgs.RedisURI
		registry.Register(&ConfigField{Name: "storage.redis_uri", Value: *GlobalCLIArgs.RedisURI, Source: SourceCLI, IsSensitive: true})
	}

	if GlobalCLIArgs.MySQLDSN != nil && *GlobalCLIArgs.MySQLDSN != "" {
		cfg.Storage.MySQLDSN = *GlobalCLIArgs.MySQLDSN
		registry.Register(&ConfigField{Name: "storage.mysql_dsn", Value: *GlobalCLIArgs.MySQLDSN, Source: SourceCLI, IsSensitive: true})
	}

	if GlobalCLIArgs.PostgresDSN != nil && *GlobalCLIArgs.PostgresDSN != "" {
		cfg.Storage.PostgresDSN = *GlobalCLIArgs.PostgresDSN
		registry.Register(&ConfigField{Name: "storage.postgres_dsn", Value: *GlobalCLIArgs.PostgresDSN, Source: SourceCLI, IsSensitive: true})
	}

	if GlobalCLIArgs.SwaggerEnabled != nil && *GlobalCLIArgs.SwaggerEnabled {
		cfg.Swagger.Enabled = *GlobalCLIArgs.SwaggerEnabled
		registry.Register(&ConfigField{Name: "swagger.enabled", Value: *GlobalCLIArgs.SwaggerEnabled, Source: SourceCLI})
	}
}

// updateRegistryFromConfig 从配置结构更新注册表
func updateRegistryFromConfig(cfg *Config, registry *ConfigRegistry, source ConfigSource) {
	registry.Register(&ConfigField{Name: "mode", Value: cfg.Mode, Source: source})
	registry.Register(&ConfigField{Name: "port", Value: cfg.Port, Source: source})
	registry.Register(&ConfigField{Name: "token", Value: cfg.Token, Source: source, IsSensitive: true})
	registry.Register(&ConfigField{Name: "log_level", Value: cfg.LogLevel, Source: source})
	registry.Register(&ConfigField{Name: "storage.driver", Value: cfg.Storage.Driver, Source: source})
	registry.Register(&ConfigField{Name: "storage.redis_uri", Value: cfg.Storage.RedisURI, Source: source, IsSensitive: true})
	registry.Register(&ConfigField{Name: "storage.mysql_dsn", Value: cfg.Storage.MySQLDSN, Source: source, IsSensitive: true})
	registry.Register(&ConfigField{Name: "storage.postgres_dsn", Value: cfg.Storage.PostgresDSN, Source: source, IsSensitive: true})
	registry.Register(&ConfigField{Name: "pagination.default_count", Value: cfg.Pagination.DefaultCount, Source: source})
	registry.Register(&ConfigField{Name: "pagination.max_count", Value: cfg.Pagination.MaxCount, Source: source})
	registry.Register(&ConfigField{Name: "swagger.enabled", Value: cfg.Swagger.Enabled, Source: source})
}

// validateConfig 验证配置有效性
func validateConfig(cfg *Config, registry *ConfigRegistry) error {
	var validationErrors []string

	// 验证端口号
	if cfg.Port != "" {
		port, err := strconv.Atoi(cfg.Port)
		if err != nil || port < 1 || port > 65535 {
			validationErrors = append(validationErrors, fmt.Sprintf("无效的端口号: %s (必须是 1-65535 之间的整数)", cfg.Port))
		}
	}

	// 验证运行模式
	validModes := []string{"debug", "test", "release"}
	if !contains(validModes, cfg.Mode) {
		validationErrors = append(validationErrors, fmt.Sprintf("无效的运行模式: %s (必须是: %v)", cfg.Mode, validModes))
	}

	// 验证日志级别
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, cfg.LogLevel) {
		validationErrors = append(validationErrors, fmt.Sprintf("无效的日志级别: %s (必须是: %v)", cfg.LogLevel, validLogLevels))
	}

	// 验证存储驱动
	validDrivers := []string{"memory", "redis", "mysql", "postgres", "authing"}
	if !contains(validDrivers, cfg.Storage.Driver) {
		validationErrors = append(validationErrors, fmt.Sprintf("无效的存储驱动: %s (必须是: %v)", cfg.Storage.Driver, validDrivers))
	}

	// 验证存储驱动特定的配置
	switch cfg.Storage.Driver {
	case "redis":
		if cfg.Storage.RedisURI == "" {
			validationErrors = append(validationErrors, "使用 Redis 存储时必须提供 Redis URI")
		}
	case "mysql":
		if cfg.Storage.MySQLDSN == "" {
			validationErrors = append(validationErrors, "使用 MySQL 存储时必须提供 MySQL DSN")
		}
	case "postgres":
		if cfg.Storage.PostgresDSN == "" {
			validationErrors = append(validationErrors, "使用 PostgreSQL 存储时必须提供 PostgreSQL DSN")
		}
	}

	// 验证分页配置
	if cfg.Pagination.DefaultCount < 1 {
		validationErrors = append(validationErrors, "分页默认数量必须大于 0")
	}
	if cfg.Pagination.MaxCount < 1 {
		validationErrors = append(validationErrors, "分页最大数量必须大于 0")
	}
	if cfg.Pagination.DefaultCount > cfg.Pagination.MaxCount {
		validationErrors = append(validationErrors, "分页默认数量不能大于最大数量")
	}

	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "; "))
	}

	return nil
}

// decryptSensitiveFields 解密敏感配置字段
func decryptSensitiveFields(cfg *Config, registry *ConfigRegistry) error {
	encryptionKey := os.Getenv("SCIM_ENCRYPTION_KEY")
	if encryptionKey == "" {
		// 如果没有加密密钥，保持原样
		return nil
	}

	// 解密 token
	if strings.HasPrefix(cfg.Token, "enc:") {
		decrypted, err := decrypt(strings.TrimPrefix(cfg.Token, "enc:"), encryptionKey)
		if err != nil {
			return fmt.Errorf("解密 token 失败: %w", err)
		}
		cfg.Token = decrypted
	}

	// 解密 Redis URI
	if strings.HasPrefix(cfg.Storage.RedisURI, "enc:") {
		decrypted, err := decrypt(strings.TrimPrefix(cfg.Storage.RedisURI, "enc:"), encryptionKey)
		if err != nil {
			return fmt.Errorf("解密 Redis URI 失败: %w", err)
		}
		cfg.Storage.RedisURI = decrypted
	}

	// 解密 MySQL DSN
	if strings.HasPrefix(cfg.Storage.MySQLDSN, "enc:") {
		decrypted, err := decrypt(strings.TrimPrefix(cfg.Storage.MySQLDSN, "enc:"), encryptionKey)
		if err != nil {
			return fmt.Errorf("解密 MySQL DSN 失败: %w", err)
		}
		cfg.Storage.MySQLDSN = decrypted
	}

	// 解密 PostgreSQL DSN
	if strings.HasPrefix(cfg.Storage.PostgresDSN, "enc:") {
		decrypted, err := decrypt(strings.TrimPrefix(cfg.Storage.PostgresDSN, "enc:"), encryptionKey)
		if err != nil {
			return fmt.Errorf("解密 PostgreSQL DSN 失败: %w", err)
		}
		cfg.Storage.PostgresDSN = decrypted
	}

	return nil
}

// encrypt 加密数据
func encrypt(plaintext, key string) (string, error) {
	block, err := aes.NewCipher([]byte(padKey(key)))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt 解密数据
func decrypt(ciphertext, key string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(padKey(key)))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("密文太短")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// padKey 填充密钥到合适长度
func padKey(key string) string {
	const keySize = 32
	if len(key) >= keySize {
		return key[:keySize]
	}
	return key + strings.Repeat("0", keySize-len(key))
}

// contains 检查字符串是否在切片中
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// PrintConfigSummary 打印配置摘要（隐藏敏感信息）
func PrintConfigSummary(cfg *Config, registry *ConfigRegistry) {
	log.Println("=============================================")
	log.Println("配置加载完成")
	log.Println("=============================================")

	fields := registry.GetAll()
	for name, field := range fields {
		value := field.Value
		if field.IsSensitive && value != "" {
			value = "***隐藏***"
		}
		log.Printf("  %s: %v (来源: %s)", name, value, field.Source)
	}

	log.Println("=============================================")
}

// GenerateConfigTemplate 生成配置文件模板
func GenerateConfigTemplate(format string) (string, error) {
	template := map[string]interface{}{
		"mode":     "debug",
		"port":     "8080",
		"token":    "your-secret-token-here",
		"log_level": "info",
		"storage": map[string]interface{}{
			"driver":       "memory",
			"redis_uri":    "redis://localhost:6379/0",
			"mysql_dsn":    "user:password@tcp(localhost:3306)/scim?charset=utf8mb4&parseTime=True&loc=Local",
			"postgres_dsn": "host=localhost user=postgres password=postgres dbname=scim port=5432 sslmode=disable",
		},
		"pagination": map[string]interface{}{
			"default_count":  20,
			"max_count":      100,
			"cursor_support": true,
		},
		"scim": map[string]interface{}{
			"default_schema": "urn:ietf:params:scim:schemas:core:2.0:User",
			"group_schema":   "urn:ietf:params:scim:schemas:core:2.0:Group",
			"error_schema":   "urn:ietf:params:scim:api:messages:2.0:Error",
			"list_schema":    "urn:ietf:params:scim:api:messages:2.0:ListResponse",
		},
		"swagger": map[string]interface{}{
			"enabled": true,
			"path":    "/swagger/*any",
		},
	}

	switch strings.ToLower(format) {
	case "yaml", "yml":
		data, err := yaml.Marshal(template)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "json":
		data, err := json.MarshalIndent(template, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("不支持的格式: %s", format)
	}
}
