ALTER TABLE entries RENAME COLUMN volume TO volume_m3;

ALTER TABLE entries 
ALTER COLUMN vehicle_id TYPE INTEGER USING vehicle_id::integer;

ALTER TABLE entries 
ALTER COLUMN status TYPE SMALLINT;