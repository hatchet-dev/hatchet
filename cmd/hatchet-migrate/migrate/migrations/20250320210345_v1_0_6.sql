-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION get_refill_value(rate_limit "RateLimit")
RETURNS INTEGER AS $$
DECLARE
    refill_amount INTEGER;
BEGIN
    IF (NOW() - rate_limit."lastRefill") >= (rate_limit."window"::INTERVAL - INTERVAL '10 milliseconds') THEN
        refill_amount := rate_limit."limitValue";
    ELSE
        refill_amount := rate_limit."value";
    END IF;
    RETURN refill_amount;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION get_refill_value(rate_limit "RateLimit")
RETURNS INTEGER AS $$
DECLARE
    refill_amount INTEGER;
BEGIN
    IF (NOW() - rate_limit."lastRefill") >= (rate_limit."window"::INTERVAL) THEN
        refill_amount := rate_limit."limitValue";
    ELSE
        refill_amount := rate_limit."value";
    END IF;
    RETURN refill_amount;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
