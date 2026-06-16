package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/joho/godotenv"
)

var cfg Config

func main() {
	godotenv.Load()
	cfg = LoadConfig()

	if cfg.MercadoPagoAccessToken == "" {
		log.Fatal("MERCADO_PAGO_ACCESS_TOKEN es obligatorio")
	}

	db, err := InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Error inicializando DB: %v", err)
	}
	defer db.Close()

	mp := NewMPClient(cfg.MercadoPagoAccessToken)
	h := NewHandlers(db, mp)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", h.Health)
	mux.HandleFunc("GET /api/numbers", h.ListNumbers)
	mux.HandleFunc("POST /api/payments/create", h.CreatePayment)
	mux.HandleFunc("GET /api/payments/{paymentId}", h.GetPaymentStatus)
	mux.HandleFunc("POST /api/webhook/mercadopago", h.Webhook)

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("GET /", fs)

	addr := cfg.Addr()
	log.Printf("Servidor iniciado en http://%s", addr)
	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		log.Fatalf("Error iniciando servidor: %v", err)
	}
}

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
