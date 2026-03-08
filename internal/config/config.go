package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	QR         QRConfig
	Kafka      KafkaConfig
	OTP        OTPConfig
	Environment string
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Host        string
	Port        string
	User        string
	Password    string
	Name        string
	SSLMode     string
	DatabaseURL string // DATABASE_URL (Railway); if set, used instead of individual fields
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	URL      string // REDIS_URL (Railway); if set, used instead of Host/Port
}

type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

type QRConfig struct {
	EncryptionKey    string
	SignatureSecret  string
}

type KafkaConfig struct {
	Brokers      []string
	TopicQRScans string
	TopicOrders  string
	TopicInventory string
}

type OTPConfig struct {
	ExpiryMinutes int
	Length        int
}

var AppConfig *Config

func Load() error {
	// Carica .env se esiste (non obbligatorio in produzione)
	_ = godotenv.Load()

	// PORT is set by Railway; SERVER_PORT for local
	serverPort := getEnv("PORT", getEnv("SERVER_PORT", "8080"))

	AppConfig = &Config{
		Server: ServerConfig{
			Port: serverPort,
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			DatabaseURL: getEnv("DATABASE_URL", getEnv("POSTGRES_URL", "")),
			Host:        getEnv("DB_HOST", "localhost"),
			Port:        getEnv("DB_PORT", "5432"),
			User:        getEnv("DB_USER", "marvaron_user"),
			Password:    getEnv("DB_PASSWORD", "marvaron_password"),
			Name:        getEnv("DB_NAME", "marvaron_db"),
			SSLMode:     getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", ""),
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "change-me-in-production"),
			ExpiryHours: getEnvAsInt("JWT_EXPIRY_HOURS", 24),
		},
		QR: QRConfig{
			EncryptionKey:   getEnv("QR_ENCRYPTION_KEY", "change-me-32-byte-key-here!!"),
			SignatureSecret: getEnv("QR_SIGNATURE_SECRET", "change-me-signature-secret"),
		},
		Kafka: KafkaConfig{
			Brokers:        parseKafkaBrokers(getEnv("KAFKA_BROKERS", "localhost:9092")),
			TopicQRScans:   getEnv("KAFKA_TOPIC_QR_SCANS", "qr-scans"),
			TopicOrders:    getEnv("KAFKA_TOPIC_ORDERS", "orders"),
			TopicInventory: getEnv("KAFKA_TOPIC_INVENTORY", "inventory"),
		},
		OTP: OTPConfig{
			ExpiryMinutes: getEnvAsInt("OTP_EXPIRY_MINUTES", 10),
			Length:        getEnvAsInt("OTP_LENGTH", 6),
		},
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	// Validazione configurazione critica
	if AppConfig.JWT.Secret == "change-me-in-production" && AppConfig.Environment == "production" {
		return fmt.Errorf("JWT_SECRET deve essere configurato in produzione")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func parseKafkaBrokers(brokers string) []string {
	// Assume comma-separated list
	result := []string{}
	parts := []rune(brokers)
	current := ""
	for _, char := range parts {
		if char == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	if len(result) == 0 {
		return []string{"localhost:9092"}
	}
	return result
}

func (c *Config) GetDSN() string {
	if c.Database.DatabaseURL != "" {
		return c.Database.DatabaseURL
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

func (c *Config) GetRedisAddr() string {
	if c.Redis.URL != "" {
		return c.Redis.URL // Caller should use URL when set
	}
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}

// UseRedisURL returns true when REDIS_URL is set (e.g. on Railway).
func (c *Config) UseRedisURL() bool {
	return c.Redis.URL != ""
}

func (c *Config) GetJWTExpiry() time.Duration {
	return time.Duration(c.JWT.ExpiryHours) * time.Hour
}
