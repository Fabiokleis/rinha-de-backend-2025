CREATE UNLOGGED TABLE payments (
    correlation_id UUID PRIMARY KEY,
    amount DECIMAL NOT NULL,
    requested_at TIMESTAMP NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
);

CREATE INDEX payments_requested_at ON payments (requested_at);
CREATE INDEX idx_payments_pending_jobs ON payments (requested_at)
WHERE status = 'pending';

ALTER TABLE payments SET (autovacuum_vacuum_scale_factor = 0.05);

CREATE OR REPLACE FUNCTION fn_notify_new_payment()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('payments_queue', NEW.correlation_id::text);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_after_insert_on_payments
AFTER INSERT ON payments
FOR EACH ROW
EXECUTE FUNCTION fn_notify_new_payment();
