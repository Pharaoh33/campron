package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Addr            string `mapstructure:"addr"`
		BaseURL         string `mapstructure:"base_url"`
		CorsAllowOrigin string `mapstructure:"cors_allow_origin"`
	} `mapstructure:"server"`

	Storage struct {
		DownloadDir string `mapstructure:"download_dir"`
	} `mapstructure:"storage"`

	Cambridge struct {
		BaseHost  string `mapstructure:"base_host"`
		UserAgent string `mapstructure:"user_agent"`
	} `mapstructure:"cambridge"`
}

func MustLoad() *Config {
	v := viper.New()

	// file config
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	// env override: CAMPRON_SERVER_ADDR, CAMPRON_STORAGE_DOWNLOAD_DIR, etc.
	v.SetEnvPrefix("CAMPRON")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// defaults
	v.SetDefault("server.addr", ":8080")
	v.SetDefault("server.base_url", "http://localhost:8080")
	v.SetDefault("server.cors_allow_origin", "*")
	v.SetDefault("storage.download_dir", "../downloads")
	v.SetDefault("cambridge.base_host", "https://dictionary.cambridge.org")

	_ = v.ReadInConfig() // ignore error; env+defaults may be enough

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	return &cfg
}
