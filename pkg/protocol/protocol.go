package protocol

// postgres notify/listen protocol

type Topic string

const (
	Payments Topic = "payments_queue"
)

type Payment struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}
