package protocol

import "time"

// postgres notify/listen protocol

type Topic string

const (
	Payments Topic = "payments_queue"
)

type Payment struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

type ProcessingPayment struct {
	*Payment
	RequestedAt time.Time `json:"requestedAt"`
}
