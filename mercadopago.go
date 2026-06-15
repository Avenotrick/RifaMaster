package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type MPClient struct {
	accessToken string
	httpClient  *http.Client
}

func NewMPClient(accessToken string) *MPClient {
	return &MPClient{
		accessToken: accessToken,
		httpClient:  &http.Client{},
	}
}

func (c *MPClient) CreatePreference(number int, buyerName, buyerEmail string, amount float64, paymentID string) (*CreatePaymentResponse, error) {
	body := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"title":        fmt.Sprintf("Rifa - Número #%d", number),
				"description":  fmt.Sprintf("Participación en la rifa - Número %d", number),
				"quantity":     1,
				"currency_id":  "ARS",
				"unit_price":   amount,
			},
		},
		"external_reference": paymentID,
		"back_urls": map[string]string{
			"success": fmt.Sprintf("%s/pago-exitoso", cfg.FrontendURL),
			"failure": fmt.Sprintf("%s/pago-fallido", cfg.FrontendURL),
			"pending": fmt.Sprintf("%s/pago-pendiente", cfg.FrontendURL),
		},
		"auto_return":       "approved",
		"notification_url":  fmt.Sprintf("%s/api/webhook/mercadopago", cfg.FrontendURL),
		"statement_descriptor": "RifaMaster",
	}

	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", "https://api.mercadopago.com/checkout/preferences", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error llamando a MP: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MP error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID              string `json:"id"`
		InitPoint       string `json:"init_point"`
		SandboxInitPoint string `json:"sandbox_init_point"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta MP: %w", err)
	}

	initPoint := result.InitPoint
	if initPoint == "" {
		initPoint = result.SandboxInitPoint
	}

	return &CreatePaymentResponse{
		PaymentID:    paymentID,
		PreferenceID: result.ID,
		InitPoint:    initPoint,
	}, nil
}

func (c *MPClient) GetPaymentInfo(paymentID string) (*MPPayment, error) {
	url := fmt.Sprintf("https://api.mercadopago.com/v1/payments/%s", paymentID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MP error obteniendo pago %s: %d", paymentID, resp.StatusCode)
	}

	var payment MPPayment
	if err := json.NewDecoder(resp.Body).Decode(&payment); err != nil {
		return nil, err
	}

	return &payment, nil
}
