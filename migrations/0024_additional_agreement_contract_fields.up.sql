ALTER TABLE additional_agreements ADD COLUMN IF NOT EXISTS delivery_conditions TEXT;
ALTER TABLE additional_agreements ADD COLUMN IF NOT EXISTS delivery_term_days  INT;
ALTER TABLE additional_agreements ADD COLUMN IF NOT EXISTS return_term_days    INT;
ALTER TABLE additional_agreements ADD COLUMN IF NOT EXISTS subject             TEXT;