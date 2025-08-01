CREATE UNLOGGED TABLE payments (
    correlation_id UUID PRIMARY KEY,
    amount DECIMAL NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    service TEXT, 
    processed_at TIMESTAMPTZ
);

CREATE INDEX payments_requested_at ON payments (requested_at);
CREATE INDEX idx_payments_pending_jobs ON payments (requested_at)
WHERE status = 'pending';

CREATE INDEX idx_payments_status_service_requested_at ON payments (status, service, requested_at);

-- one row table to store median value
CREATE UNLOGGED TABLE processing_metrics (
    metric_name TEXT PRIMARY KEY,
    metric_value BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Pre-populate it with a starting value
INSERT INTO processing_metrics (metric_name, metric_value, updated_at) 
VALUES ('rolling_average', 0, NOW());

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

-- procedure to update one row table and return it
CREATE OR REPLACE FUNCTION update_rolling_payment_average()
RETURNS BIGINT AS $$
DECLARE
    new_metric_value BIGINT;
BEGIN
    INSERT INTO processing_metrics (metric_name, metric_value, updated_at)
    SELECT
        'rolling_average',
        ROUND(COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY amount), 0))::BIGINT,
        NOW()
    FROM (
        SELECT amount
        FROM payments
        WHERE status = 'completed'
        ORDER BY processed_at DESC
        LIMIT 10
    ) AS latest_payments
    ON CONFLICT (metric_name) DO UPDATE
    SET
        metric_value = EXCLUDED.metric_value,
        updated_at = EXCLUDED.updated_at
    RETURNING metric_value INTO new_metric_value;
    
    RETURN new_metric_value;
END;
$$ LANGUAGE plpgsql;
