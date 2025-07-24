package listener

import (
	"context"
	"errors"
	"fmt"
	db "rinha/internal/database"
	"time"

	prot "rinha/pkg/protocol"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func processPayment(id uint64, p *prot.Payment) {
	fmt.Printf("[ID: %v] processing %+v\n", id, p)
}

// returns error waiting notification if timeout exceeds
func claimPaymentOrder(conn *pgxpool.Conn, timeout time.Duration) (*prot.Payment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	not, err := conn.Conn().WaitForNotification(ctx)
	if err != nil {
		return nil, err
	}

	//fmt.Printf("[ID: %v][PID: %v][CHANNEL: %v] payload: %+v\n", id, not.PID, not.Channel, not.Payload)
	tx, err := conn.Begin(db.PgxCtx)
	var p prot.Payment

	// try to claim notification payment pending order
	err = tx.QueryRow(db.PgxCtx, `
                UPDATE payments
		SET status = 'processing'
		WHERE correlation_id = $1 AND status = 'pending'
		RETURNING correlation_id, amount`,
		not.Payload,
	).Scan(&p.CorrelationId, &p.Amount)

	if err == nil {
		if err := tx.Commit(db.PgxCtx); err != nil {
			tx.Rollback(db.PgxCtx)
			return nil, fmt.Errorf("unexpected error on commit specific job: %w", err)
		}
		return &p, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("unexpected error claiming specific job: %w", err)
	}

	// in case of another listener grabs notification order first
	// claim latest notification payment peding order
	err = tx.QueryRow(ctx, `
                UPDATE payments
		SET status = 'processing'
		WHERE correlation_id = (
			SELECT correlation_id
			FROM payments
			WHERE status = 'pending'
			ORDER BY requested_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING correlation_id, amount`,
	).Scan(&p.CorrelationId, &p.Amount)

	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("unexpected error claiming specific job: %w", err)
		}
		return &p, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		_ = tx.Commit(ctx)
		return nil, nil
	}

	return nil, fmt.Errorf("unexpected error claiming specific job: %w", err)
}

// start listening and processing new notifications
func processPaymentsQueue(ctx context.Context, id uint64, topic string) error {
	//services := ctx.Value("services").(*PaymentServices)

	conn, err := db.Pgxpool.Acquire(db.PgxCtx)
	if err != nil {
		return err
	}
	_, err = conn.Exec(db.PgxCtx, fmt.Sprintf("LISTEN %s", topic))
	if err != nil {
		return err
	}

	defer conn.Release()
	fmt.Printf("[ID: %v][TOPIC: %v] waiting notifications\n", id, topic)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("stop processing topic %v\n", topic)
			_, err = conn.Exec(db.PgxCtx, fmt.Sprintf("UNLISTEN %s", topic))
			return err
		default:
			p, err := claimPaymentOrder(conn, 100*time.Millisecond)
			if err != nil {
				if (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) && ctx.Err() == nil {
					//fmt.Println(err.Error())
					//fmt.Println("listen timeout")
					continue
				}
				return err
			}
			if p != nil {
				processPayment(id, p)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// subscribe all handlers
func assignTopics() {
	l.subscribe(18, "payments_queue", processPaymentsQueue)
}
