package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	DBType      string
	DBURL       string
	OllamaURL   string
	OllamaModel string
	Language    string
	AIBrainPath string
}

// AppDataDir returns ~/Library/Application Support/Kuliss on macOS.
// This is the canonical location for app data when running from /Applications.
func AppDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dir := filepath.Join(home, "Library", "Application Support", "Kuliss")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// DataPath returns an absolute path inside AppDataDir.
func DataPath(filename string) string {
	return filepath.Join(AppDataDir(), filename)
}

func Load() *Config {
	// Priority: app data dir >> executable dir >> cwd (dev mode)
	envPaths := []string{
		DataPath(".env"),
	}
	if execPath, err := os.Executable(); err == nil {
		envPaths = append(envPaths, filepath.Join(filepath.Dir(execPath), ".env"))
	}
	envPaths = append(envPaths, ".env")

	for _, p := range envPaths {
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Overload(p)
			break
		}
	}

	cfg := &Config{
		DBType:      getEnv("DB_TYPE", "sqlite"),
		DBURL:       getEnv("DB_URL", "messages.db"), // will be made absolute below if needed
		OllamaURL:   getEnv("OLLAMA_URL", "http://localhost:11434"),
		OllamaModel: getEnv("OLLAMA_MODEL", "gemma4:e4b"),
		Language:    getEnv("LANGUAGE", "en"),
		AIBrainPath: getEnv("AI_BRAIN_PATH", ""),
	}

	// Resolve AI_BRAIN path: env > app data dir > executable dir > cwd
	if cfg.AIBrainPath == "" {
		cfg.AIBrainPath = DataPath("AI_BRAIN")
		if _, err := os.Stat(cfg.AIBrainPath); os.IsNotExist(err) {
			if execPath, err := os.Executable(); err == nil {
				exeDir := filepath.Join(filepath.Dir(execPath), "AI_BRAIN")
				if _, err := os.Stat(exeDir); err == nil {
					cfg.AIBrainPath = exeDir
				}
			}
		}
	}
	if _, err := os.Stat(cfg.AIBrainPath); os.IsNotExist(err) {
		if cwd, err := os.Getwd(); err == nil {
			cwdPath := filepath.Join(cwd, "AI_BRAIN")
			if _, err := os.Stat(cwdPath); err == nil {
				cfg.AIBrainPath = cwdPath
			}
		}
	}

	if cfg.DBType == "sqlite" && !filepath.IsAbs(cfg.DBURL) {
		cfg.DBURL = DataPath(cfg.DBURL)
	}

	return cfg
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("zorunlu env eksik: %s", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
