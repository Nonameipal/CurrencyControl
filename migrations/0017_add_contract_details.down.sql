ALTER TABLE contracts 
    DROP COLUMN IF EXISTS contract_name,
    DROP COLUMN IF EXISTS delivery_date,
    DROP COLUMN IF EXISTS delivery_conditions,
    DROP COLUMN IF EXISTS delivery_term_days,
    DROP COLUMN IF EXISTS return_term_days,
    DROP COLUMN IF EXISTS receiver_name,
    DROP COLUMN IF EXISTS receiver_account,
    DROP COLUMN IF EXISTS receiver_country;
