package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
	"github.com/joho/godotenv"
)

type Config struct {
	siteName          string
	siteNameW1        string
	siteNameW2        string
	siteNameOrangeWord int
	siteLogoInner     string
	baseURL  string
	mnemonic string
	apiKey   string

	port                    string
	network                 spark.Network
	storageDirectory        string
	randomBytesLength       int
	redeemFeeBPS            int64
	minRedeemFeeMsat        int64
	minFundAmountMsat       int64
	maxFundAmountMsat       int64
	minRedeemAmountMsat     int64
	maxVoucherExpireSeconds int64
	maxVouchersPerBatch     int64
	createActive            bool
	fundActive              bool
	redeemActive            bool
	refundActive                bool
	refundWorkerIntervalSeconds int64
	batchEnabled                bool
	invoiceExpirySeconds    int64

	// Optional features — silently disabled if empty
	adminToken     string // Bearer token for /admin/stats
	githubURL      string // Shown in About modal and footer
	defaultDialCode string // Default dial code for phone number field, e.g. +27
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

	cfg.siteName = os.Getenv("SITE_NAME")
	if cfg.siteName == "" {
		return nil, errMissingEnv("SITE_NAME")
	}
	words := strings.Fields(cfg.siteName)
	if len(words) != 2 {
		return nil, fmt.Errorf("SITE_NAME must be exactly two words, got %q", cfg.siteName)
	}
	cfg.siteNameW1 = words[0]
	cfg.siteNameW2 = words[1]

	orangeWordStr := os.Getenv("SITE_NAME_ORANGE_WORD")
	if orangeWordStr == "" {
		orangeWordStr = "1"
	}
	cfg.siteNameOrangeWord, err = strconv.Atoi(orangeWordStr)
	if err != nil || (cfg.siteNameOrangeWord != 1 && cfg.siteNameOrangeWord != 2) {
		return nil, fmt.Errorf("SITE_NAME_ORANGE_WORD must be 1 or 2")
	}
	if cfg.siteNameOrangeWord == 1 {
		cfg.siteLogoInner = cfg.siteNameW1 + `<span>` + cfg.siteNameW2 + `</span>`
	} else {
		cfg.siteLogoInner = `<span>` + cfg.siteNameW1 + `</span>` + cfg.siteNameW2
	}

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

	if v := os.Getenv("RANDOM_BYTES_LENGTH"); v == "" {
		return nil, errMissingEnv("RANDOM_BYTES_LENGTH")
	} else {
		var err error
		if cfg.randomBytesLength, err = strconv.Atoi(v); err != nil {
			return nil, err
		}
		if cfg.randomBytesLength < 1 || cfg.randomBytesLength > 32 {
			return nil, fmt.Errorf("RANDOM_BYTES_LENGTH must be between 1 and 32")
		}
	}

	if v := os.Getenv("REDEEM_FEE_BPS"); v == "" {
		return nil, errMissingEnv("REDEEM_FEE_BPS")
	} else {
		var err error
		if cfg.redeemFeeBPS, err = strconv.ParseInt(v, 10, 64); err != nil {
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

	if v := os.Getenv("MIN_FUND_AMOUNT_MSAT"); v == "" {
		return nil, errMissingEnv("MIN_FUND_AMOUNT_MSAT")
	} else {
		var err error
		cfg.minFundAmountMsat, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
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

	if cfg.minFundAmountMsat >= cfg.maxFundAmountMsat {
		return nil, fmt.Errorf("MIN_FUND_AMOUNT_MSAT must be less than MAX_FUND_AMOUNT_MSAT")
	}

	if v := os.Getenv("MIN_REDEEM_AMOUNT_MSAT"); v == "" {
		return nil, errMissingEnv("MIN_REDEEM_AMOUNT_MSAT")
	} else {
		var err error
		if cfg.minRedeemAmountMsat, err = strconv.ParseInt(v, 10, 64); err != nil {
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
	cfg.batchEnabled, _ = strconv.ParseBool(os.Getenv("BATCH_ENABLED"))

	cfg.refundWorkerIntervalSeconds = int64(86400)
	if v := os.Getenv("REFUND_WORKER_INTERVAL_SECONDS"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("REFUND_WORKER_INTERVAL_SECONDS must be a positive integer")
		}
		cfg.refundWorkerIntervalSeconds = parsed
	}

	if v := os.Getenv("INVOICE_EXPIRY_SECONDS"); v == "" {
		return nil, errMissingEnv("INVOICE_EXPIRY_SECONDS")
	} else {
		cfg.invoiceExpirySeconds, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	cfg.adminToken = os.Getenv("ADMIN_TOKEN")
	cfg.githubURL = os.Getenv("GITHUB_URL")
	cfg.defaultDialCode = os.Getenv("DEFAULT_DIAL_CODE")
	if cfg.defaultDialCode == "" {
		cfg.defaultDialCode = "+27"
	}

	return &Server{cfg: cfg}, nil
}
