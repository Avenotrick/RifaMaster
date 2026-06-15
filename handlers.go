package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

type Handlers struct {
	db *sql.DB
	mp *MPClient
}

func NewHandlers(db *sql.DB, mp *MPClient) *Handlers {
	return &Handlers{db: db, mp: mp}
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "1.0.0",
	})
}

func (h *Handlers) ListNumbers(w http.ResponseWriter, r *http.Request) {
	numbers, err := GetAllNumbers(h.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error al obtener números")
		return
	}

	var public []NumberPublic
	var available, reserved, sold int

	for _, n := range numbers {
		p := NumberPublic{
			Number: n.Number,
			Status: n.Status,
		}
		if n.Status == "sold" {
			p.BuyerName = n.BuyerName
			sold++
		} else if n.Status == "reserved" {
			reserved++
		} else {
			available++
		}
		public = append(public, p)
	}

	writeJSON(w, http.StatusOK, NumbersResponse{
		Numbers: public,
		Counts: Counts{
			Available: available,
			Reserved:  reserved,
			Sold:      sold,
			Total:     100,
		},
	})
}

func (h *Handlers) GetNumber(w http.ResponseWriter, r *http.Request) {
	numStr := r.PathValue("number")
	num := parseInt(numStr)
	if num < 1 || num > 100 {
		writeError(w, http.StatusBadRequest, "Número inválido")
		return
	}

	n, err := GetNumber(h.db, num)
	if err != nil {
		writeError(w, http.StatusNotFound, "Número no encontrado")
		return
	}

	p := NumberPublic{Number: n.Number, Status: n.Status}
	if n.Status == "sold" {
		p.BuyerName = n.BuyerName
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handlers) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	if req.Number < 1 || req.Number > 100 {
		writeError(w, http.StatusBadRequest, "Número debe estar entre 1 y 100")
		return
	}
	if req.BuyerName == "" || req.BuyerEmail == "" {
		writeError(w, http.StatusBadRequest, "Nombre y email obligatorios")
		return
	}

	n, err := GetNumber(h.db, req.Number)
	if err != nil {
		writeError(w, http.StatusNotFound, "Número no encontrado")
		return
	}
	if n.Status != "available" {
		writeError(w, http.StatusConflict, "El número no está disponible")
		return
	}

	paymentID := newUUID()
	amount := 1000.0

	pref, err := h.mp.CreatePreference(req.Number, req.BuyerName, req.BuyerEmail, amount, paymentID)
	if err != nil {
		log.Printf("Error MP: %v", err)
		writeError(w, http.StatusBadGateway, "Error al procesar el pago")
		return
	}

	if err := ReserveNumber(h.db, req.Number, req.BuyerName, req.BuyerEmail, paymentID); err != nil {
		writeError(w, http.StatusConflict, "El número ya no está disponible")
		return
	}

	payment := &Payment{
		ID:           paymentID,
		PreferenceID: pref.PreferenceID,
		Number:       req.Number,
		Status:       "pending",
		BuyerName:    req.BuyerName,
		BuyerEmail:   req.BuyerEmail,
		Amount:       amount,
	}
	if err := CreatePayment(h.db, payment); err != nil {
		log.Printf("Error guardando pago: %v", err)
	}

	writeJSON(w, http.StatusCreated, pref)
}

func (h *Handlers) GetPaymentStatus(w http.ResponseWriter, r *http.Request) {
	paymentID := r.PathValue("paymentId")

	p, err := GetPayment(h.db, paymentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Pago no encontrado")
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (h *Handlers) Webhook(w http.ResponseWriter, r *http.Request) {
	var hook MPWebhook
	if err := json.NewDecoder(r.Body).Decode(&hook); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	log.Printf("Webhook recibido: action=%s type=%s", hook.Action, hook.Type)

	paymentID := hook.ID
	if hook.Data != nil && hook.Data.ID != "" {
		paymentID = hook.Data.ID
	}

	if paymentID == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	mpPayment, err := h.mp.GetPaymentInfo(paymentID)
	if err != nil {
		log.Printf("Error obteniendo info del pago %s: %v", paymentID, err)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	extRef := mpPayment.ExternalReference
	if extRef == "" {
		log.Printf("Webhook sin external_reference, ignorando: %s", paymentID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	switch mpPayment.Status {
	case "approved":
		payment, err := GetPayment(h.db, extRef)
		if err != nil {
			log.Printf("Error obteniendo pago local %s: %v", extRef, err)
			break
		}

		if err := ConfirmPayment(h.db, extRef); err != nil {
			log.Printf("Error confirmando pago %s: %v", extRef, err)
			break
		}

		log.Printf("Pago aprobado: %s (ref: %s, número #%d)", paymentID, extRef, payment.Number)

		if err := sendConfirmationEmail(payment.BuyerEmail, payment.BuyerName, payment.Number); err != nil {
			log.Printf("Error enviando email a %s: %v", payment.BuyerEmail, err)
		}

	case "rejected", "cancelled", "refunded", "charged_back":
		if err := RejectPayment(h.db, extRef); err != nil {
			log.Printf("Error rechazando pago %s: %v", extRef, err)
		} else {
			log.Printf("Pago rechazado: %s (ref: %s)", paymentID, extRef)
		}

	default:
		log.Printf("Estado no manejado: %s - %s", mpPayment.Status, paymentID)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) AvailableNumbers(w http.ResponseWriter, r *http.Request) {
	numbers, err := GetAllNumbers(h.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error al obtener números")
		return
	}

	var available []int
	for _, n := range numbers {
		if n.Status == "available" {
			available = append(available, n.Number)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"available": available,
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
