ALTER TABLE elections
    ADD COLUMN description TEXT NOT NULL DEFAULT '',
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD CONSTRAINT elections_type_check
        CHECK (type IN ('parliamentary', 'presidential', 'mixed')),
    ADD CONSTRAINT elections_status_check
        CHECK (status IN ('draft', 'active', 'finished', 'cancelled')),
    ADD CONSTRAINT elections_period_check
        CHECK (starts_at < ends_at);

CREATE TABLE election_ballots (
    id          BIGSERIAL PRIMARY KEY,
    election_id BIGINT NOT NULL REFERENCES elections (id) ON DELETE CASCADE,
    ballot_type TEXT NOT NULL,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT election_ballots_type_check
        CHECK (ballot_type IN ('party', 'candidate')),
    CONSTRAINT election_ballots_unique_type_per_election
        UNIQUE (election_id, ballot_type)
);

CREATE TABLE ballot_options (
    id               BIGSERIAL PRIMARY KEY,
    ballot_id        BIGINT NOT NULL REFERENCES election_ballots (id) ON DELETE CASCADE,
    source_entity_id BIGINT NOT NULL,
    display_name     TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    image_url        TEXT NOT NULL DEFAULT '',
    snapshot_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    position         INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT ballot_options_unique_source_per_ballot
        UNIQUE (ballot_id, source_entity_id),
    CONSTRAINT ballot_options_id_ballot_unique
        UNIQUE (id, ballot_id)
);

CREATE TABLE votes (
    id               BIGSERIAL PRIMARY KEY,
    ballot_id        BIGINT NOT NULL,
    ballot_option_id BIGINT NOT NULL,
    user_id          BIGINT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT votes_ballot_fk
        FOREIGN KEY (ballot_id)
        REFERENCES election_ballots (id)
        ON DELETE CASCADE,
    CONSTRAINT votes_ballot_option_fk
        FOREIGN KEY (ballot_option_id, ballot_id)
        REFERENCES ballot_options (id, ballot_id)
        ON DELETE RESTRICT,
    CONSTRAINT votes_one_user_vote_per_ballot
        UNIQUE (ballot_id, user_id)
);

CREATE INDEX election_ballots_election_id_idx
    ON election_ballots (election_id);

CREATE INDEX ballot_options_ballot_id_idx
    ON ballot_options (ballot_id);

CREATE INDEX votes_user_id_idx
    ON votes (user_id);

CREATE INDEX votes_ballot_option_id_idx
    ON votes (ballot_option_id);
