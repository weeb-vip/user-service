package config

import (
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/jinzhu/configor"
)

type Config struct {
	APPConfig          AppConfig
	DBConfig           DBConfig
	RefreshTokenConfig RefreshTokenConfig
	KafkaConfig        KafkaConfig
	MinioConfig        MinioConfig
}

type AppConfig struct {
	Name                      string `env:"CONFIG__APP_CONFIG__NAME" required:"true" default:"card-delivery-service"`
	APPName                   string `env:"CONFIG__APP_CONFIG__APP_NAME" default:"user-service"`
	Version                   string `env:"APP__VERSION" default:"local"`
	Env                       string `env:"APP_ENV" default:"dev"`
	Port                      int    `env:"CONFIG__APP_CONFIG__PORT" default:"3002"`
	KeyRollingDurationInHours int    `env:"CONFIG__APP_CONFIG__KEY_ROLLING_DURATION_IN_HOURS" default:"1"`
	InternalGraphQLURL        string `env:"INTERNAL_GRAPHQL_URL" default:"http://localhost:5001/graphql"`
	JWTValiditySeconds        int    `env:"CONFIG__APP_CONFIG__JWT_VALIDITY_SECONDS" default:"900"` // 15 minutes.
}

type DBConfig struct {
	Host               string `env:"DBHOST" required:"true" default:"localhost"`
	Port               uint   `env:"DBPORT" required:"true" default:"5432"`
	User               string `env:"DBUSER" required:"true" default:"postgres"`
	Password           string `env:"DBPASSWORD" required:"true" default:"mysecretpassword"`
	DB                 string `env:"DBNAME" required:"true" default:"auth"`
	SSL                string `env:"DBSSL" default:"false"`
	MigrationTableName string `env:"DBMIGRATIONTABLE" default:"__migrations_auth"`
}

type RefreshTokenConfig struct {
	TokenTTL int `env:"CONFIG__REFRESH_TOKEN_CONFIG__TOKEN_TTL" default:"4380"` // 6 months in hours.
}

type KafkaConfig struct {
	ConsumerGroupName string `default:"user-service-group" env:"KAFKA_CONSUMER_GROUP_NAME"`
	BootstrapServers  string `default:"localhost:9092" env:"KAFKA_BOOTSTRAP_SERVERS"`
	Offset            string `default:"earliest" env:"KAFKA_OFFSET"`
	Topic             string `default:"user-created" env:"KAFKA_TOPIC"`
	ProducerTopic     string `default:"nil" env:"KAFKA_PRODUCER_TOPIC"`
}

type MinioConfig struct {
	Endpoint        string `default:"localhost:9000" env:"MINIO_ENDPOINT"`
	AccessKeyID     string `default:"minio" env:"MINIO_ACCESS_KEY_ID"`
	SecretAccessKey string `default:"minio123" env:"MINIO_SECRET_ACCESS_KEY"`
	UseSSL          bool   `default:"false" env:"MINIO_USESSL"`
	Bucket          string `default:"anime" env:"MINIO_BUCKET"`
}

func LoadConfig() (*Config, error) {
	var config Config
	err := configor.
		New(&configor.Config{AutoReload: false}).
		Load(&config, fmt.Sprintf("%s/config.%s.json", getConfigLocation(), getEnv()))

	if err != nil {
		return nil, err
	}

	return &config, nil
}

func LoadConfigOrPanic() *Config {
	config, err := LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	return config
}

func getConfigLocation() string {
	_, filename, _, _ := runtime.Caller(0) // nolint

	return path.Join(path.Dir(filename), "../config")
}

func getEnv() string {
	prod := "prod"
	dev := "dev"
	docker := "docker"

	val := os.Getenv("APP_ENV")
	switch val {
	case prod:
		return prod
	case docker:
		return docker
	default:
		return dev
	}
}
