package config

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var cfg Config
var doOnce sync.Once

type Config struct {
	Application struct {
		Port  string `mapstructure:"PORT"`
		Group string `mapstructure:"GROUP"`
	} `mapstructure:"APPLICATION"`
	DB struct {
		Postgre struct {
			Host    string `mapstructure:"DB_HOST"`
			Port    int    `mapstructure:"DB_PORT"`
			Name    string `mapstructure:"DB_NAME"`
			User    string `mapstructure:"DB_USERNAME"`
			Pass    string `mapstructure:"DB_PASSWORD"`
			Migrate bool   `mapstructure:"MIGRATE"`
			Params  string `mapstructure:"DB_PARAMS"`
		} `mapstructure:"POSTGRE"`
	} `mapstructure:"DB"`
	Auth struct {
		SecretKey               string `mapstructure:"SECRET_KEY"`
		AccessTokenExpiredTime  int    `mapstructure:"ACCESS_TOKEN_EXPIRED_TIME"`
		RefreshTokenExpiredTime string `mapstructure:"REFRESH_TOKEN_EXPIRED_TIME"`
		BcryptSalt              int    `mapstructure:"BCRYPT_SALT"`
	} `mapstructure:"AUTH"`
}

func Get() Config {
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		log.WithLevel(zerolog.FatalLevel).Msg(fmt.Sprintf("cannot read .yaml file: %v", err))
	}

	doOnce.Do(func() {
		err := viper.Unmarshal(&cfg)
		if err != nil {
			log.WithLevel(zerolog.FatalLevel).Msg("cannot unmarshaling config")
		}
	})

	return cfg
}
