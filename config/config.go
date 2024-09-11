package config

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"goflare.io/ember"
	emberConfig "goflare.io/ember/config"
	"goflare.io/ignite"
	"goflare.io/payment/driver"
)

const (
	ServerStartPort = ":8080"
)

type Config struct {
	Stripe   StripeConfig
	Postgres PostgresConfig
	Redis    RedisConfig
}

type StripeConfig struct {
	SecretKey string `mapstructure:"secret_key"`
}

type PostgresConfig struct {
	URL string `mapstructure:"url"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
}

func ProvideApplicationConfig() (*Config, error) {

	viper.SetConfigFile("./config.yaml")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func ProvidePostgresConn(appConfig *Config) (driver.PostgresPool, error) {

	conn, err := driver.ConnectSQL(appConfig.Postgres.URL)
	if err != nil {
		return nil, err
	}

	return conn.Pool, nil
}

func ProvideEmber(appConfig *Config) (*ember.MultiCache, error) {

	conn, err := driver.ConnectRedis(appConfig.Redis.Addr, appConfig.Redis.Password, 0)
	if err != nil {
		return nil, err
	}

	config := emberConfig.NewConfig()
	cache, err := ember.NewMultiCache(context.Background(), &config, conn)
	if err != nil {
		log.Println(fmt.Errorf("failed to create cache: %w", err))
		return nil, err
	}

	return cache, nil
}

func ProvideIgnite() ignite.Manager {
	return ignite.NewManager()
}

func NewLogger() *zap.Logger {

	logger, _ := zap.NewProduction()
	return logger
}
