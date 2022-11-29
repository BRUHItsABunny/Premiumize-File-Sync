package utils

import (
	"fmt"
	"github.com/joho/godotenv"
)

func LoadEnv() error {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("INFO: Failed to load .env")
	} else {
		fmt.Println("INFO: Loaded .env")
	}
	return err
}
