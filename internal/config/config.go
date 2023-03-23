package config

import (
	"fmt"
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
		WalletMnemonic:    getEnv("WALLET_MNEMONIC", ""),
		ChainID:           getEnv("CHAIN_ID", ""),
		ChainRPC:          getEnv("CHAIN_RPC", ""),
		ChainGRPC:         getEnv("CHAIN_GRPC", ""),
		AuraPoolBackend:   getEnv("AURA_POOL_BACKEND", ""),
		StateFile:         getEnv("STATE_FILE", ""),
		MaxRetries:        getEnvAsInt("MAX_RETRIES", 10),
		RetryInterval:     getEnvAsDuration("RETRY_INTERVAL", time.Second*30),
		RelayInterval:     getEnvAsDuration("RELAY_INTERVAL", time.Second*5),
		PaymentDenom:      getEnv("PAYMENT_DENOM", "acudos"),
		Port:              getEnvAsInt("PORT", 3000),
		PrettyLogging:     getEnvAsInt("PRETTY_LOGGING", 0),
		EmailFrom:         getEnv("EMAIL_FROM", ""),
		ServiceEmail:      getEnv("SERVICE_EMAIL", ""),
		SendgridApiKey:    getEnv("SENDGRID_API_KEY", ""),
		AuraPoolApiKey:    getEnv("AURA_POOL_API_KEY", ""),
		EmailSendInterval: getEnvAsDuration("EMAIL_SEND_INTERVAL", time.Minute*30),
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
	WalletMnemonic    string
	ChainID           string
	ChainRPC          string
	ChainGRPC         string
	AuraPoolBackend   string
	StateFile         string
	MaxRetries        int
	RetryInterval     time.Duration
	RelayInterval     time.Duration
	PaymentDenom      string
	Port              int
	PrettyLogging     int
	EmailFrom         string
	ServiceEmail      string
	SendgridApiKey    string
	AuraPoolApiKey    string
	EmailSendInterval time.Duration
}

func (cfg *Config) HasPrettyLogging() bool {
	return cfg.PrettyLogging == 1
}

func (cfg *Config) HasValidEmailConfig() bool {
	return cfg.SendgridApiKey != "" && cfg.EmailFrom != "" && cfg.ServiceEmail != ""
}

func (cfg *Config) String() string {
	return fmt.Sprintf("Config { WalletMnemonic(Hidden for security), ChainID(%s), ChainRPC(%s), ChainGRPC(%s), AuraPoolBackend(%s), StateFile(%s), MaxRetries(%d), RetryInterval(%d), RelayInterval(%d), PaymentDenom(%s), Port(%d) PrettyLogging(%d) SendgridApiKey(%s) EmailFrom(%s) ServiceEmail(%s) EmailSendInterval(%d)}", cfg.ChainID, cfg.ChainRPC, cfg.ChainGRPC, cfg.AuraPoolBackend, cfg.StateFile, cfg.MaxRetries, cfg.RetryInterval, cfg.RelayInterval, cfg.PaymentDenom, cfg.Port, cfg.PrettyLogging, "Hidden for security", cfg.EmailFrom, cfg.ServiceEmail, cfg.EmailSendInterval)
}
