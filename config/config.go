package config

import (
	"os"

	// process .env
	"github.com/joho/godotenv"
)

func Load() map[string]string {
	env := os.Getenv("GIFKA_ENV")
	if env == "" {
		env = "development"
	}

	var environment map[string]string
	environment, err := godotenv.Read("config/.env." + env)

	if (err != nil) {
		panic(err)
	}

	return environment
}
