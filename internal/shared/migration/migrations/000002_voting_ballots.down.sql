DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS ballot_options;
DROP TABLE IF EXISTS election_ballots;

ALTER TABLE elections
    DROP CONSTRAINT IF EXISTS elections_period_check,
    DROP CONSTRAINT IF EXISTS elections_status_check,
    DROP CONSTRAINT IF EXISTS elections_type_check,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at,
    DROP COLUMN IF EXISTS description;
