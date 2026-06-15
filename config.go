package main

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL              string
	Host                     string
	Port                     int
	FrontendURL              string
	MercadoPagoAccessToken   string
	MercadoPagoWebhookSecret string
	SmtpHost                 string
	SmtpPort                 int
	SmtpUser                 string
	SmtpPass                 string
	SmtpFrom                 string
}

func LoadConfig() Config {
	port, _ := strconv.Atoi(getEnv("PORT", "3000"))
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))

	return Config{
		DatabaseURL:              getEnv("DATABASE_URL", "file:rifa.db?cache=shared"),
		Host:                     getEnv("HOST", "0.0.0.0"),
		Port:                     port,
		FrontendURL:              getEnv("FRONTEND_URL", "http://localhost:3000"),
		MercadoPagoAccessToken:   os.Getenv("MERCADO_PAGO_ACCESS_TOKEN"),
		MercadoPagoWebhookSecret: os.Getenv("MERCADO_PAGO_WEBHOOK_SECRET"),
		SmtpHost:                 getEnv("SMTP_HOST", "smtp.gmail.com"),
		SmtpPort:                 smtpPort,
		SmtpUser:                 os.Getenv("SMTP_USER"),
		SmtpPass:                 os.Getenv("SMTP_PASS"),
		SmtpFrom:                 getEnv("SMTP_FROM", os.Getenv("SMTP_USER")),
	}
}

func (c Config) SmtpEnabled() bool {
	return c.SmtpHost != "" && c.SmtpUser != "" && c.SmtpPass != ""
}

func (c Config) Addr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
