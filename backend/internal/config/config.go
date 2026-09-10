package config

import (
	"encoding/json"
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
	ETAProfiles           map[string]float64
}

type jsonConfig struct {
	ETAProfiles map[string]float64 `json:"eta_profiles"`
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	fileConfig := loadJSONConfig()
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
		ETAProfiles:           fileConfig.ETAProfiles,
	}
}

func loadJSONConfig() jsonConfig {
	configPath := filepath.Join("internal", "config", "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read JSON config %s: %v", configPath, err)
	}

	var fileConfig jsonConfig
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		log.Fatalf("Failed to parse JSON config %s: %v", configPath, err)
	}

	requiredProfiles := []string{"urban", "suburban", "rural", "highway", "mixed"}
	for _, profile := range requiredProfiles {
		multiplier, exists := fileConfig.ETAProfiles[profile]
		if !exists || multiplier <= 0 {
			log.Fatalf("Invalid or missing ETA profile multiplier: %s", profile)
		}
	}

	return fileConfig
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
