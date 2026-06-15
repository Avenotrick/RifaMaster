package main

type Number struct {
	ID        int64  `json:"id"`
	Number    int    `json:"number"`
	Status    string `json:"status"`
	BuyerName string `json:"buyer_name,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type NumberPublic struct {
	Number    int    `json:"number"`
	Status    string `json:"status"`
	BuyerName string `json:"buyer_name,omitempty"`
}

type Payment struct {
	ID            string  `json:"id"`
	PreferenceID  string  `json:"preference_id,omitempty"`
	Number        int     `json:"number"`
	Status        string  `json:"status"`
	BuyerName     string  `json:"buyer_name"`
	BuyerEmail    string  `json:"buyer_email"`
	Amount        float64 `json:"amount"`
	CreatedAt     string  `json:"created_at"`
}

type NumbersResponse struct {
	Numbers []NumberPublic `json:"numbers"`
	Counts  Counts         `json:"counts"`
}

type Counts struct {
	Available int `json:"available"`
	Reserved  int `json:"reserved"`
	Sold      int `json:"sold"`
	Total     int `json:"total"`
}

type CreatePaymentRequest struct {
	Number    int    `json:"number"`
	BuyerName string `json:"buyer_name"`
	BuyerEmail string `json:"buyer_email"`
}

type CreatePaymentResponse struct {
	PaymentID    string `json:"payment_id"`
	PreferenceID string `json:"preference_id"`
	InitPoint    string `json:"init_point"`
}

type MPWebhook struct {
	Action string    `json:"action"`
	Data   *MPData   `json:"data"`
	ID     string    `json:"id,omitempty"`
	Type   string    `json:"type,omitempty"`
}

type MPData struct {
	ID string `json:"id"`
}

type MPPayment struct {
	ID                int64       `json:"id"`
	Status            string      `json:"status"`
	ExternalReference string      `json:"external_reference"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
