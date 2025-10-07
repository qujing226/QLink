package config

import (
	"os"
	"strconv"
	"time"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	DID      DIDConfig      `json:"did"`
	Security SecurityConfig `json:"security"`
	Log      LogConfig      `json:"log"`
	CORS     CORSConfig     `json:"cors"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// DIDConfig DID相关配置
type DIDConfig struct {
	NodeURL string `json:"node_url"`
	Method  string `json:"method"`
	Prefix  string `json:"prefix"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	JWTSecret     string        `json:"jwt_secret"`
	JWTExpiration time.Duration `json:"jwt_expiration"`
	RateLimit     int           `json:"rate_limit"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level string `json:"level"`
	File  string `json:"file"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	Origins []string      `json:"origins"`
	MaxAge  time.Duration `json:"max_age"`
}

// Load 加载配置
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv(EnvServerHost, DefaultServerHost),
			Port: getEnv(EnvServerPort, DefaultServerPort),
		},
		Database: DatabaseConfig{
			Type: DefaultDatabaseType,
			URL:  getEnv(EnvDatabaseURL, DefaultDatabasePath),
		},
		DID: DIDConfig{
			NodeURL: getEnv("DID_NODE_URL", "http://localhost:8080"),
			Method:  DefaultDIDMethod,
			Prefix:  DefaultDIDPrefix,
		},
		Security: SecurityConfig{
			JWTSecret:     getEnv(EnvJWTSecret, "your-secret-key"),
			JWTExpiration: DefaultJWTExpiration,
			RateLimit:     getEnvInt(EnvRateLimit, DefaultRateLimit),
		},
		Log: LogConfig{
			Level: getEnv(EnvLogLevel, DefaultLogLevel),
			File:  getEnv(EnvLogFile, DefaultLogFile),
		},
		CORS: CORSConfig{
			Origins: []string{"*"}, // 在生产环境中应该配置具体的域名
			MaxAge:  DefaultCORSMaxAge,
		},
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// IsDevelopment 判断是否为开发环境
func (c *Config) IsDevelopment() bool {
	return getEnv(EnvEnvironment, EnvDevelopment) == EnvDevelopment
}

// IsProduction 判断是否为生产环境
func (c *Config) IsProduction() bool {
	return getEnv(EnvEnvironment, EnvDevelopment) == EnvProduction
}

// GetServerAddress 获取服务器地址
func (c *Config) GetServerAddress() string {
	return c.Server.Host + ":" + c.Server.Port
}