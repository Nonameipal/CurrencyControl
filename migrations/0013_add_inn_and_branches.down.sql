ALTER TABLE contracts 
    DROP COLUMN IF EXISTS client_id,
    DROP COLUMN IF EXISTS branch_id;

ALTER TABLE counterparties 
    DROP COLUMN IF EXISTS inn;

DROP TABLE IF EXISTS branches;
