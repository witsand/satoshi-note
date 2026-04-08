package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
	"github.com/joho/godotenv"
)

type Config struct {
	baseURL  string
	mnemonic string
	apiKey   string

	port                        string
	network                     spark.Network
	storageDirectory            string
	redeemFeeBPS                int64
	internalFeeBPS              int64
	minInternalFeeMsat          int64
	minRedeemFeeMsat            int64
	maxFundAmountMsat           int64
	maxVoucherExpireSeconds     int64
	maxVouchersPerBatch         int64
	createActive                bool
	fundActive                  bool
	redeemActive                bool
	refundActive                bool
	refundWorkerIntervalSeconds int64
	paymentCooldown             time.Duration
	invoiceExpirySeconds        int64
}

func errMissingEnv(name string) error {
	return fmt.Errorf("missing environment variable: %s", name)
}

func loadConfig() (*Server, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	cfg := &Config{}

	cfg.baseURL = os.Getenv("BASE_URL")
	if cfg.baseURL == "" {
		return nil, errMissingEnv("BASE_URL")
	}

	cfg.mnemonic = os.Getenv("MNEMONIC")
	if cfg.mnemonic == "" {
		return nil, errMissingEnv("MNEMONIC")
	}

	cfg.apiKey = os.Getenv("BREEZ_API_KEY")
	if cfg.apiKey == "" {
		return nil, errMissingEnv("BREEZ_API_KEY")
	}

	cfg.port = os.Getenv("PORT")
	if cfg.port == "" {
		return nil, errMissingEnv("PORT")
	}

	networkStr := os.Getenv("NETWORK")
	switch networkStr {
	case "mainnet":
		cfg.network = spark.NetworkMainnet
	default:
		cfg.network = spark.NetworkRegtest
	}

	cfg.storageDirectory = os.Getenv("STORAGE_DIRECTORY")
	if cfg.storageDirectory == "" {
		return nil, errMissingEnv("STORAGE_DIRECTORY")
	}

	if v := os.Getenv("REDEEM_FEE_BPS"); v == "" {
		return nil, errMissingEnv("REDEEM_FEE_BPS")
	} else {
		var err error
		if cfg.redeemFeeBPS, err = strconv.ParseInt(v, 10, 64); err != nil {
			return nil, err
		}
	}

	if v := os.Getenv("INTERNAL_FEE_BPS"); v == "" {
		return nil, errMissingEnv("INTERNAL_FEE_BPS")
	} else {
		var err error
		if cfg.internalFeeBPS, err = strconv.ParseInt(v, 10, 64); err != nil {
			return nil, err
		}
	}

	if v := os.Getenv("MIN_INTERNAL_FEE_MSAT"); v == "" {
		return nil, errMissingEnv("MIN_INTERNAL_FEE_MSAT")
	} else {
		var err error
		if cfg.minInternalFeeMsat, err = strconv.ParseInt(v, 10, 64); err != nil {
			return nil, err
		}
	}

	if v := os.Getenv("MIN_REDEEM_FEE_MSAT"); v == "" {
		return nil, errMissingEnv("MIN_REDEEM_FEE_MSAT")
	} else {
		var err error
		if cfg.minRedeemFeeMsat, err = strconv.ParseInt(v, 10, 64); err != nil {
			return nil, err
		}
	}

	if v := os.Getenv("MAX_FUND_AMOUNT_MSAT"); v == "" {
		return nil, errMissingEnv("MAX_FUND_AMOUNT_MSAT")
	} else {
		var err error
		if cfg.maxFundAmountMsat, err = strconv.ParseInt(v, 10, 64); err != nil {
			return nil, err
		}
	}

	if v := os.Getenv("MAX_VOUCHER_EXPIRE_SECONDS"); v == "" {
		return nil, errMissingEnv("MAX_VOUCHER_EXPIRE_SECONDS")
	} else {
		cfg.maxVoucherExpireSeconds, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	if v := os.Getenv("MAX_VOUCHERS_PER_BATCH"); v == "" {
		return nil, errMissingEnv("MAX_VOUCHERS_PER_BATCH")
	} else {
		cfg.maxVouchersPerBatch, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	cfg.createActive, _ = strconv.ParseBool(os.Getenv("CREATE_ACTIVE"))
	cfg.fundActive, _ = strconv.ParseBool(os.Getenv("FUND_ACTIVE"))
	cfg.redeemActive, _ = strconv.ParseBool(os.Getenv("REDEEM_ACTIVE"))
	cfg.refundActive, _ = strconv.ParseBool(os.Getenv("REFUND_ACTIVE"))

	cfg.refundWorkerIntervalSeconds = int64(86400)
	if v := os.Getenv("REFUND_WORKER_INTERVAL_SECONDS"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("REFUND_WORKER_INTERVAL_SECONDS must be a positive integer")
		}
		cfg.refundWorkerIntervalSeconds = parsed
	}

	cfg.paymentCooldown = 1000 * time.Millisecond
	if v := os.Getenv("PAYMENT_COOLDOWN_MS"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil || parsed < 0 {
			return nil, fmt.Errorf("PAYMENT_COOLDOWN_MS must be a non-negative integer")
		}
		cfg.paymentCooldown = time.Duration(parsed) * time.Millisecond
	}

	if v := os.Getenv("INVOICE_EXPIRY_SECONDS"); v == "" {
		return nil, errMissingEnv("INVOICE_EXPIRY_SECONDS")
	} else {
		cfg.invoiceExpirySeconds, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	return &Server{cfg: cfg}, nil
}
