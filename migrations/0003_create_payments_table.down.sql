DROP TRIGGER  IF EXISTS trg_payments_overdue_days ON payments;
DROP FUNCTION IF EXISTS calc_overdue_days();
DROP TABLE    IF EXISTS payments;
