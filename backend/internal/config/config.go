package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort            string
	DatabaseURL           string
	MediaDir              string
	JWTSecret             string
	JWTRefreshSecret      string
	JWTAccessTokenExpiry  int64
	JWTRefreshTokenExpiry int64
	AdminUser             string
	AdminPass             string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	mediaDir := resolveMediaDir(getEnv("MEDIA_DIR", "../local-store/media"))

	return &Config{
		ServerPort:            getEnv("PORT", "8080"),
		DatabaseURL:           getEnv("DATABASE_URL", ""),
		MediaDir:              mediaDir,
		JWTSecret:             getRequiredEnv("JWT_SECRET"),
		JWTRefreshSecret:      getRequiredEnv("JWT_REFRESH_SECRET"),
		JWTAccessTokenExpiry:  getEnvAsInt64("JWT_ACCESS_TOKEN_EXPIRY", 3600),
		JWTRefreshTokenExpiry: getEnvAsInt64("JWT_REFRESH_TOKEN_EXPIRY", 604800),
		AdminUser:             getRequiredEnv("SUPERADMIN_USERNAME"),
		AdminPass:             getRequiredEnv("SUPERADMIN_PASSWORD"),
	}
}

func resolveMediaDir(pathValue string) string {
	if pathValue == "" {
		return ""
	}

	absPath, err := filepath.Abs(pathValue)
	if err != nil {
		log.Printf("Failed to resolve MEDIA_DIR as absolute path (%s): %v", pathValue, err)
		return pathValue
	}

	return absPath
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Required environment variable missing: %s", key)
	}

	return value
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Fatalf("Invalid integer value for %s: %s", key, value)
	}

	return parsed
}
