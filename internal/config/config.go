package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

func NewConfig(envPath string) (Config, error) {
	if err := godotenv.Load(envPath); err != nil {
		return Config{}, err
	}

	return Config{
		WalletMnemonic:  getEnv("WALLET_MNEMONIC", ""),
		ChainID:         getEnv("CHAIN_ID", ""),
		ChainRPC:        getEnv("CHAIN_RPC", ""),
		ChainGRPC:       getEnv("CHAIN_GRPC", ""),
		AuraPoolBackend: getEnv("AURA_POOL_BACKEND", ""),
		StateFile:       getEnv("STATE_FILE", ""),
		MaxRetries:      getEnvAsInt("MAX_RETRIES", 10),
		RetryInterval:   getEnvAsDuration("RETRY_INTERVAL", time.Second*30),
		RelayInterval:   getEnvAsDuration("RELAY_INTERVAL", time.Second*5),
		PaymentDenom:    getEnv("PAYMENT_DENOM", "acudos"),
	}, nil
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultVal
}

func getEnvAsDuration(name string, defaultVal time.Duration) time.Duration {
	valStr := getEnv(name, "")
	if valStr == "" {
		return defaultVal
	}
	if duration, err := time.ParseDuration(valStr); err == nil {
		return duration
	}
	return defaultVal
}

type Config struct {
	WalletMnemonic  string
	ChainID         string
	ChainRPC        string
	ChainGRPC       string
	AuraPoolBackend string
	StateFile       string
	MaxRetries      int
	RetryInterval   time.Duration
	RelayInterval   time.Duration
	PaymentDenom    string
}
