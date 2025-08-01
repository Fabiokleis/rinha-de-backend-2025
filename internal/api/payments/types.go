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
	*p.ProcessingPayment
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

type Service struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

type PaymentSummaryRow struct {
	Name   string
	Metric Service
}

type SummaryResponse struct {
	Default  Service `json:"default"`
	Fallback Service `json:"fallback"`
}

func (sr *SummaryResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
