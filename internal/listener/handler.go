package listener

import (
	"context"
	"fmt"
	db "rinha/internal/database"
	prot "rinha/pkg/protocol"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func waitAndProcessPayment(conn *pgxpool.Conn) error {
	not, err := conn.Conn().WaitForNotification(db.PgxCtx)
	if err != nil {
		return err
	}
	payload, err := prot.Decode(not.Payload)
	if err != nil {
		return err
	}
	fmt.Printf("[%v][%v] %+v\n", not.PID, not.Channel, payload)
	return nil
}

func processPaymentsQueue(ctx context.Context, topic string) error {
	conn, err := db.Pgxpool.Acquire(db.PgxCtx)
	if err != nil {
		return err
	}
	_, err = conn.Exec(db.PgxCtx, fmt.Sprintf("LISTEN %s", topic))
	if err != nil {
		return err
	}

	defer conn.Release()
	fmt.Printf("topic %v waiting notifications\n", topic)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("stop processing topic %v\n", topic)
			conn.Exec(db.PgxCtx, fmt.Sprintf("UNLISTEN %s", topic))
			return nil
		default:
			if err := waitAndProcessPayment(conn); err != nil {
				return err
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func assignTopics() {
	l.subscribe(string(prot.Payments), processPaymentsQueue)
}
