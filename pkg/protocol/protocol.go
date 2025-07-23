package protocol

// postgres notify/listen protocol

type Topic string

const (
	Payments Topic = "payments_queue"
)

// topic encoded data
type Payload struct {
	buffer []byte
}

type Codec[T any] interface {
	Encode() ([]byte, error)
	Decode(data string) (T, error)
}
