package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

func NewConfig(configPath string) (Config, error) {
	config := Config{}

	file, err := os.Open(configPath)
	if err != nil {
		return config, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)

	if err := d.Decode(&config); err != nil {
		return config, err
	}

	return config, nil
}

type Config struct {
	WalletMnemonic string `yaml:"wallet_mnemonic"`
	Chain          struct {
		ID   string `yaml:"id"`
		RPC  string `yaml:"rpc"`
		GRPC string `yaml:"grpc"`
	} `yaml:"chain"`
	TokenisedInfraUrl string        `yaml:"tokenised_infra_url"`
	StateFile         string        `yaml:"state_file"`
	MaxRetries        int           `yaml:"max_retries"`
	RetryInterval     time.Duration `yaml:"retry_interval"`
	RelayInterval     time.Duration `yaml:"relay_interval"`
	PaymentDenom      string        `yaml:"payment_denom"`
}
