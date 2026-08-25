ALTER TABLE contracts
    ADD COLUMN IF NOT EXISTS total_amount DECIMAL(18, 2) NOT NULL DEFAULT 0;

UPDATE contracts SET total_amount = remaining_amount WHERE total_amount = 0;

CREATE OR REPLACE FUNCTION manage_contract_balance()
RETURNS TRIGGER AS $$
DECLARE
    v_remaining DECIMAL(18, 2);
    v_old_amount DECIMAL(18, 2) := 0;
BEGIN
    IF TG_OP = 'DELETE' THEN
        UPDATE contracts
        SET remaining_amount = remaining_amount + OLD.amount,
            updated_at = NOW()
        WHERE id = OLD.contract_id;
        RETURN OLD;
    END IF;

    IF TG_OP = 'UPDATE' THEN
        v_old_amount := OLD.amount;
    END IF;

    SELECT remaining_amount INTO v_remaining
    FROM contracts
    WHERE id = NEW.contract_id
    FOR UPDATE;

    IF (NEW.amount - v_old_amount) > v_remaining THEN
        RAISE EXCEPTION
            'Сумма платежа (%) превышает остаток по контракту (%). Требуется дополнительное соглашение.',
            NEW.amount, v_remaining;
    END IF;

    UPDATE contracts
    SET remaining_amount = remaining_amount - (NEW.amount - v_old_amount),
        updated_at = NOW()
    WHERE id = NEW.contract_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_payments_contract_balance
    AFTER INSERT OR UPDATE OR DELETE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION manage_contract_balance();
