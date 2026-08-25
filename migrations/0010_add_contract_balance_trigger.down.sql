DROP TRIGGER  IF EXISTS trg_payments_contract_balance ON payments;
DROP FUNCTION IF EXISTS manage_contract_balance();
ALTER TABLE contracts DROP COLUMN IF EXISTS total_amount;
