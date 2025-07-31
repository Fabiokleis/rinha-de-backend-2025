package listener

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	db "rinha/internal/database"
	prot "rinha/pkg/protocol"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var processed atomic.Uint64
var median atomic.Uint64

// watch and update median value based on total of processed payments
func processedWatcher(ctx context.Context, id uint64, topic string) error {
	conn, err := db.Pgxpool.Acquire(db.PgxCtx)
	if err != nil {
		return err
	}
	defer conn.Release()

	fmt.Printf("[ID: %v][TOPIC: %v] waiting notifications\n", id, topic)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("stop processing topic %v\n", topic)
			return nil
		default:
			nProcessed := processed.Load()
			if nProcessed > 0 && nProcessed%10 == 0 {
				fmt.Printf("processed count reached %d. Triggering rolling average recalculation.\n", nProcessed)
				var latestMedian uint64
				err := conn.QueryRow(ctx, "SELECT update_rolling_payment_average();").Scan(&latestMedian)
				if err != nil {
					return err
				}
				fmt.Printf("[ID: %v][TOPIC: %v] median: %+v\n", id, topic, latestMedian)
				median.Store(latestMedian)

			}
			time.Sleep(100 * time.Millisecond)
		}

	}
}

// send payment to payment processor
func processPayment(services *PaymentServices, id uint64, conn *pgxpool.Conn, p *prot.ProcessingPayment) error {
	fmt.Printf("[ID: %v] processing %v\n", id, p.CorrelationId)

	latestMedian := median.Load()

	service := "default"
	processorUrl := *services.defaultUrl
	if latestMedian < uint64(p.Amount) {
		processorUrl = *services.fallbackUrl
		service = "fallback"
	}

	body, _ := json.Marshal(map[string]interface{}{
		"correlationId": p.CorrelationId,
		"amount":        p.Amount,
		"requestedAt":   p.RequestedAt,
	})

	resp, err := http.Post(processorUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = conn.Exec(db.PgxCtx, `
                     UPDATE payments
                     SET status = 'completed', processed_at = NOW(), service = $1
                     WHERE correlation_id = $2`, service, p.CorrelationId)
	if err != nil {
		return err
	}

	processed.Add(1)
	fmt.Printf("[ID: %v][MEDIAN: %v] processed %+v\n", id, latestMedian, p)
	return nil
}

// returns error waiting notification if timeout exceeds
func claimPaymentOrder(conn *pgxpool.Conn, timeout time.Duration) (*prot.ProcessingPayment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	not, err := conn.Conn().WaitForNotification(ctx)
	if err != nil {
		return nil, err
	}

	//fmt.Printf("[ID: %v][PID: %v][TOPIC: %v] payload: %+v\n", id, not.PID, not.Channel, not.Payload)
	tx, err := conn.Begin(db.PgxCtx)
	p := prot.ProcessingPayment{Payment: &prot.Payment{}}

	// try to claim notification payment pending order
	err = tx.QueryRow(db.PgxCtx, `
                UPDATE payments
		SET status = 'processing'
		WHERE correlation_id = $1 AND status = 'pending'
		RETURNING correlation_id, amount, requested_at`,
		not.Payload,
	).Scan(&p.CorrelationId, &p.Amount, &p.RequestedAt)

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
		RETURNING correlation_id, amount, requested_at`,
	).Scan(&p.CorrelationId, &p.Amount, &p.RequestedAt)

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
	services := ctx.Value("services").(*PaymentServices)

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
				processPayment(services, id, conn, p)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// subscribe all handlers
func assignTopics() {
	l.subscribe(1, "processed_watcher", processedWatcher)
	l.subscribe(18, "payments_queue", processPaymentsQueue)
	//l.subscribe(1, "health", healthChecker)
}
