package listener

import (
	"context"
	"errors"
	"fmt"
	db "rinha/internal/database"
	prot "rinha/pkg/protocol"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// returns error wating notification on timeout
func waitAndProcessPayment(conn *pgxpool.Conn, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	not, err := conn.Conn().WaitForNotification(ctx)
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

// start listening and processing new notifications
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
			_, err = conn.Exec(db.PgxCtx, fmt.Sprintf("UNLISTEN %s", topic))
			return err
		default:
			if err := waitAndProcessPayment(conn, 100*time.Millisecond); err != nil {
				if (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) && ctx.Err() == nil {
					//fmt.Println(err.Error())
					//fmt.Println("listen timeout")
					continue
				}
				return err
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// subscribe all handlers
func assignTopics() {
	l.subscribe(string(prot.Payments), processPaymentsQueue)
}
