package payments

import (
	"errors"
	"net/http"
	p "rinha/pkg/protocol"
	"time"
)

type PaymentRequest struct {
	*p.Payment
}

type PaymentProcessRequest struct {
	*p.Payment
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
	*p.Payment
	RequestedAt time.Time `json:"requestedAt"`
	Status      string    `json:"status"`
}

func (pr *PaymentResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
