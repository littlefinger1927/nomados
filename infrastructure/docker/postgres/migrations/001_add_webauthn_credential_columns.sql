-- Migration: Add WebAuthn credential columns to devices table
-- These columns store the credential data from WebAuthn registration ceremonies.

ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS credential_id BYTEA,
    ADD COLUMN IF NOT EXISTS credential_public_key BYTEA,
    ADD COLUMN IF NOT EXISTS sign_count BIGINT DEFAULT 0;