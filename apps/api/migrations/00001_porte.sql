-- +goose Up
-- porte owns sign-in: the accounts, the identity links and the sessions.
-- These are the tables of porte/pg Schema at v0.19.0 that its UserStore,
-- IdentityStore and SessionStore read, in their final shape: the columns porte
-- adds through guarded ALTERs for older databases are written inline, since
-- this database starts empty.
--
-- Left out on purpose: porte_login_codes and porte_magic_links (porte/pg treats
-- both as optional and gare has no CLI login and no magic link), and
-- porte_api_keys (gare issues no API keys). Apps, env and deploys stay on disk.
--
-- porte_users is porte's own table rather than a gare users table, so the
-- daemon uses porte/pg's UpsertFromOIDC as is instead of re-implementing its
-- account matching and takeover rules. Do not rename a column: porte/pg's
-- queries are written against these names, and a divergence fails on the
-- login path rather than at boot.

CREATE TABLE porte_users (
	id             bigserial PRIMARY KEY,
	facile_id      text UNIQUE,
	email          text NOT NULL UNIQUE,
	email_verified boolean NOT NULL DEFAULT false,
	name           text NOT NULL DEFAULT '',
	avatar_url     text NOT NULL DEFAULT '',
	avatar_source  text NOT NULL DEFAULT '',
	locale         text NOT NULL DEFAULT 'en',
	created_at     timestamptz NOT NULL DEFAULT now(),
	updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE porte_identities (
	user_id         bigint NOT NULL REFERENCES porte_users(id) ON DELETE CASCADE,
	provider        text NOT NULL,
	subject         text NOT NULL,
	password_hash   text NOT NULL DEFAULT '',
	access_token    text NOT NULL DEFAULT '',
	refresh_token   text NOT NULL DEFAULT '',
	token_expiry    timestamptz,
	roles           jsonb,
	roles_synced_at timestamptz,
	synced_at       timestamptz,
	created_at      timestamptz DEFAULT now(),
	PRIMARY KEY (provider, subject)
);
CREATE INDEX porte_identities_user_idx ON porte_identities (user_id);

CREATE TABLE porte_sessions (
	id           bigserial PRIMARY KEY,
	token_hash   text NOT NULL UNIQUE,
	user_id      bigint NOT NULL REFERENCES porte_users(id) ON DELETE CASCADE,
	label        text NOT NULL DEFAULT '',
	created_at   timestamptz NOT NULL DEFAULT now(),
	last_used_at timestamptz NOT NULL DEFAULT now(),
	expires_at   timestamptz
);
CREATE INDEX porte_sessions_user_idx ON porte_sessions (user_id);
CREATE INDEX porte_sessions_expiry_idx ON porte_sessions (expires_at)
	WHERE expires_at IS NOT NULL;

-- +goose Down
-- Forward-only: roll forward with a new migration, never down.
SELECT 'porte migration is not reversible' WHERE false;
