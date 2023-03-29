package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShouldPass(t *testing.T) {
	expectedCfg := Config{
		WalletMnemonic:    "rebel wet poet torch carpet gaze axis ribbon approve depend inflict menu",
		ChainID:           "cudos-local-network",
		ChainRPC:          "http://127.0.0.1:26657",
		ChainGRPC:         "127.0.0.1:9090",
		AuraPoolBackend:   "http://127.0.0.1:8080",
		StartingHeight:    2,
		MaxRetries:        10,
		RetryInterval:     30 * time.Second,
		RelayInterval:     5 * time.Second,
		PaymentDenom:      "acudos",
		Port:              3000,
		EmailSendInterval: 30 * time.Minute,
	}

	haveCfg, err := NewConfig("../../.env.example")
	require.NoError(t, err)
	require.Equal(t, expectedCfg, haveCfg)
}

func TestShouldFailIfNotExistingFile(t *testing.T) {
	_, err := NewConfig("badpath")
	require.Error(t, err)
}

func TestShouldFailIfConfigIsInvalidEnv(t *testing.T) {
	_, err := NewConfig("./testdata/invalid_config.env")
	require.Error(t, err)
}

func TestGetEnvShouldReturnDefaultIfKeyNotFound(t *testing.T) {
	require.Equal(t, "def", getEnv(str, "def"))
}

func TestGetEnvAsInt(t *testing.T) {
	require.Equal(t, 1337, getEnvAsInt(str, 1337))

	require.NoError(t, os.Setenv(str, "1338"))
	require.Equal(t, 1338, getEnvAsInt(str, 1338))

}

func TestGetEnvAsDurationShouldReturnDefaultIfKeyNotFound(t *testing.T) {
	require.Equal(t, time.Second*1337, getEnvAsDuration(str, time.Second*1337))
}

func TestGetEnvAsDurationShouldReturnDefaultIfKeyHasInvalidValue(t *testing.T) {
	require.NoError(t, os.Setenv(str, "a"))
	require.Equal(t, time.Second*1337, getEnvAsDuration(str, time.Second*1337))
}

func TestHasPrettyLogging(t *testing.T) {
	require.True(t, (&Config{PrettyLogging: 1}).HasPrettyLogging())
	require.False(t, (&Config{PrettyLogging: 0}).HasPrettyLogging())
}

func TestHasValidEmailSettings(t *testing.T) {
	require.True(t, (&Config{SendgridApiKey: str, EmailFrom: str, ServiceEmail: str}).HasValidEmailConfig())
	require.False(t, (&Config{SendgridApiKey: "", EmailFrom: str, ServiceEmail: str}).HasValidEmailConfig())
	require.False(t, (&Config{SendgridApiKey: str, EmailFrom: "", ServiceEmail: str}).HasValidEmailConfig())
	require.False(t, (&Config{SendgridApiKey: str, EmailFrom: str, ServiceEmail: ""}).HasValidEmailConfig())
}

func TestString(t *testing.T) {
	expectedCfgString := "Config { WalletMnemonic(Hidden for security), ChainID(cudos-local-network), ChainRPC(http://127.0.0.1:26657), ChainGRPC(127.0.0.1:9090), AuraPoolBackend(http://127.0.0.1:8080), StartingHeight(2), MaxRetries(10), RetryInterval(30000000000), RelayInterval(5000000000), PaymentDenom(acudos), Port(3000) PrettyLogging(0) SendgridApiKey(Hidden for security) EmailFrom() ServiceEmail() EmailSendInterval(1800000000000)}"

	haveCfg, err := NewConfig("../../.env.example")
	require.NoError(t, err)
	require.Equal(t, expectedCfgString, haveCfg.String())
}

const str = "test"
