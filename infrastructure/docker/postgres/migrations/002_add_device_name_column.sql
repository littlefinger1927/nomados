-- Add device_name column to devices table for multi-device WebAuthn support
ALTER TABLE devices ADD COLUMN IF NOT EXISTS name TEXT DEFAULT '';
ALTER TABLE devices ADD COLUMN IF NOT EXISTS aaguid BYTEA;