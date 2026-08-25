ALTER TABLE counterparties 
    ADD COLUMN IF NOT EXISTS branch_id INT REFERENCES branches(id);
