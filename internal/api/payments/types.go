package payments

import (
	"errors"
	"net/http"
	"time"
)

type Payment struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

type PaymentRequest struct {
	*Payment
}

type PaymentProcessRequest struct {
	*Payment
	RequestedAt time.Time `json:"requestedAt"`
}

func (pay *PaymentRequest) Bind(r *http.Request) error {
	if pay.Payment == nil {
		return errors.New("missing required payment fields.")
	}
	return nil
}

func (pay *PaymentProcessRequest) Bind(r *http.Request) error {
	if pay.Payment == nil {
		return errors.New("missing required payment fields.")
	}
	return nil
}

type PaymentProcessResponse struct {
	Message string `json:"message"`
}

func (pay *PaymentProcessResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type PaymentResponse struct {
	*Payment
	RequestedAt time.Time `json:"requestedAt"`
}

func (pr *PaymentResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
