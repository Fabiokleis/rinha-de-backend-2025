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

type PaymentDetailsRequest struct {
	Id string
}

func (pay *PaymentRequest) Bind(r *http.Request) error {
	if pay.Payment == nil {
		return errors.New("missing required payment fields.")
	}
	return nil
}

func (pay *PaymentDetailsRequest) Bind(r *http.Request) error {
	if pay == nil {
		return errors.New("missing id.")
	}
	return nil
}

type PaymentResponse struct {
	*Payment
	RequestedAt time.Time `json:"requested_at"`
}

func (pr *PaymentResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
