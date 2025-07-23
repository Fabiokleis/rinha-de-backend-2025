package protocol

import (
	"encoding/json"
	"time"
)

type PaymentPayload struct {
	CorrelationId string    `json:"correlationId"`
	Amount        float64   `json:"amount"`
	RequestedAt   time.Time `json:"requestedAt"`
}

func (p *PaymentPayload) Encode() ([]byte, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func Decode(data string) (*PaymentPayload, error) {
	var decoded *PaymentPayload
	err := json.Unmarshal([]byte(data), &decoded)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}
