ALTER TABLE additional_agreements 
RENAME COLUMN new_delivery_conditions TO delivery_conditions;

ALTER TABLE additional_agreements 
RENAME COLUMN new_delivery_term_days TO delivery_term_days;

ALTER TABLE additional_agreements 
RENAME COLUMN new_return_term_days TO return_term_days;

ALTER TABLE additional_agreements 
RENAME COLUMN currency TO foreign_currency;

ALTER TABLE additional_agreements 
ALTER COLUMN foreign_currency DROP NOT NULL;

ALTER TABLE additional_agreements 
ADD COLUMN IF NOT EXISTS subject TEXT;

ALTER TABLE additional_agreements 
ALTER COLUMN agreement_number DROP NOT NULL;

ALTER TABLE additional_agreements 
ALTER COLUMN agreement_date DROP NOT NULL;
