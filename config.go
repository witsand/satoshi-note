package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	port                      string
	storageDirectory          string
	defaultRefundAfterSeconds int
	maxRefundAfterSeconds     int
	maxVouchers               int
}

func errMissingEnv(name string) error {
	return fmt.Errorf("missing environment variable: %s", name)
}

func loadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	conf := &Config{}

	conf.port = os.Getenv("PORT")
	if conf.port == "" {
		return nil, errMissingEnv("PORT")
	}

	conf.storageDirectory = os.Getenv("STORAGE_DIRECTORY")
	if conf.storageDirectory == "" {
		return nil, errMissingEnv("STORAGE_DIRECTORY")
	}

	if v := os.Getenv("DEFAULT_REFUND_AFTER_SECONDS"); v == "" {
		return nil, errMissingEnv("DEFAULT_REFUND_AFTER_SECONDS")
	} else {
		var err error
		if conf.defaultRefundAfterSeconds, err = strconv.Atoi(v); err != nil {
			return nil, err
		}
	}

	if v := os.Getenv("MAX_REFUND_AFTER_SECONDS"); v == "" {
		return nil, errMissingEnv("MAX_REFUND_AFTER_SECONDS")
	} else {
		var err error
		if conf.maxRefundAfterSeconds, err = strconv.Atoi(v); err != nil {
			return nil, err
		}
	}

	if v := os.Getenv("MAX_VOUCHERS"); v == "" {
		return nil, errMissingEnv("MAX_VOUCHERS")
	} else {
		var err error
		if conf.maxVouchers, err = strconv.Atoi(v); err != nil {
			return nil, err
		}
	}

	return conf, nil
}
