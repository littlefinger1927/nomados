/**
 * In-memory key management for NomadOS end-to-end encryption.
 *
 * The master key is derived once after login (from the user's passphrase
 * via PBKDF2) and cached in memory. It is NEVER persisted to localStorage
 * or any other durable store.
 *
 * Key hierarchy:
 *   Passphrase + Salt → PBKDF2-SHA256 → Master Key (32 bytes)
 *   Master Key + workspace_id → HKDF-SHA256 → Workspace Key (32 bytes)
 *   Workspace Key → AES-256-GCM → Encrypted File
 *
 * For Phase 5, workspace keys are derived client-side using HKDF-SHA256
 * to match the server-side derivation in vault-service/keyderivation.
 */

// ---------------------------------------------------------------------------
// In-memory master key cache (never persisted)
// ---------------------------------------------------------------------------

let cachedMasterKey: ArrayBuffer | null = null;

/** Retrieve the cached master key, or null if not available. */
export function getCachedMasterKey(): ArrayBuffer | null {
  return cachedMasterKey;
}

/** Cache the master key in memory. Overwrites any previously cached key. */
export function setCachedMasterKey(key: ArrayBuffer): void {
  cachedMasterKey = key;
}

/**
 * Clear the cached master key and attempt to zeroize the buffer.
 * Call this on logout or session expiry.
 */
export function clearCachedMasterKey(): void {
  if (cachedMasterKey) {
    // Best-effort zeroization — overwrite the buffer with zeros
    const view = new Uint8Array(cachedMasterKey);
    view.fill(0);
  }
  cachedMasterKey = null;
}

// ---------------------------------------------------------------------------
// Master key derivation (PBKDF2-SHA256)
// ---------------------------------------------------------------------------

/**
 * Derive a 32-byte master key from a passphrase and salt using PBKDF2-SHA256.
 *
 * This is the Phase 5 alternative to Argon2id (which requires WASM).
 * The parameters are chosen to match the server-side derivation where possible,
 * though the server uses Argon2id. PBKDF2 is used client-side as a
 * practical compromise until WASM-based Argon2id is available.
 *
 * @param passphrase - User's passphrase
 * @param salt - Salt for key derivation (at least 16 bytes recommended)
 * @returns 32-byte master key as ArrayBuffer
 */
export async function deriveMasterKey(
  passphrase: string,
  salt: ArrayBuffer | Uint8Array,
): Promise<ArrayBuffer> {
  const encoder = new TextEncoder();

  // Import passphrase as key material for PBKDF2
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    encoder.encode(passphrase),
    { name: 'PBKDF2' },
    false,
    ['deriveBits'],
  );

  // Ensure salt is an ArrayBuffer (Web Crypto requires BufferSource)
  const saltBuffer = salt instanceof Uint8Array ? salt.buffer as ArrayBuffer : salt;

  // Derive 256 bits (32 bytes) using PBKDF2-SHA256
  // 600,000 iterations matches OWASP 2023 recommendations for PBKDF2-SHA256
  const derivedBits = await crypto.subtle.deriveBits(
    {
      name: 'PBKDF2',
      salt: saltBuffer,
      iterations: 600_000,
      hash: 'SHA-256',
    },
    keyMaterial,
    256, // 32 bytes
  );

  return derivedBits;
}

// ---------------------------------------------------------------------------
// Workspace key derivation (HKDF-SHA256, client-side)
// ---------------------------------------------------------------------------

const WORKSPACE_KEY_INFO = new TextEncoder().encode('nomados-workspace-key');

/**
 * Derive a 32-byte workspace key from a master key and workspace ID
 * using HKDF-SHA256, matching the server-side derivation in
 * vault-service/internal/keyderivation.
 *
 * The server uses: HKDF-SHA256(masterKey, info="nomados-workspace-key", salt=workspaceID)
 * We replicate this client-side so we can encrypt/decrypt without a round-trip.
 *
 * @param masterKey - 32-byte master key
 * @param workspaceId - UUID of the workspace
 * @returns 32-byte workspace key as ArrayBuffer
 */
export async function deriveWorkspaceKey(
  masterKey: ArrayBuffer,
  workspaceId: string,
): Promise<ArrayBuffer> {
  // Import master key as key material for HKDF
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    masterKey,
    { name: 'HKDF' },
    false,
    ['deriveBits'],
  );

  const encoder = new TextEncoder();

  // HKDF-SHA256 with:
  //   salt = "nomados-workspace-key" (the info string used as salt, matching Go implementation)
  //   info = workspaceId (used as HKDF expand info, matching Go implementation)
  //   length = 256 bits (32 bytes)
  //
  // Note: The Go server uses hkdf.New(sha256.New, masterKey, workspaceKeyInfo, workspaceID)
  // where workspaceKeyInfo = []byte("nomados-workspace-key") is the HKDF salt
  // and workspaceID is the info parameter passed to hkdf expand.
  // Web Crypto's HKDF maps: salt → salt, info → info
  const derivedBits = await crypto.subtle.deriveBits(
    {
      name: 'HKDF',
      hash: 'SHA-256',
      salt: WORKSPACE_KEY_INFO,
      info: encoder.encode(workspaceId),
    },
    keyMaterial,
    256, // 32 bytes
  );

  return derivedBits;
}

// ---------------------------------------------------------------------------
// Convenience: derive and cache master key, then derive workspace key
// ---------------------------------------------------------------------------

/**
 * Derive a workspace key from a cached master key and workspace ID.
 * Returns null if no master key is cached.
 *
 * @param workspaceId - UUID of the workspace
 * @returns 32-byte workspace key, or null if master key is unavailable
 */
export async function getWorkspaceKey(
  workspaceId: string,
): Promise<ArrayBuffer | null> {
  const masterKey = getCachedMasterKey();
  if (!masterKey) return null;

  return deriveWorkspaceKey(masterKey, workspaceId);
}

/**
 * Initialize the master key from the current session's access token.
 *
 * For Phase 5, the master key is derived deterministically from the access
 * token using HKDF-SHA256 with a fixed salt. This ensures:
 * - Each session produces the same master key for the same user
 * - The master key is available without requiring a separate passphrase
 * - The master key is never persisted (only held in memory)
 *
 * In a future phase, this will be replaced with passphrase-derived keys
 * (Argon2id on desktop, PBKDF2 fallback) where the user enters a
 * passphrase after login.
 *
 * This function is idempotent — if a master key is already cached, it
 * returns immediately without re-derivation.
 *
 * @returns true if the master key is available (cached or newly derived)
 */
export async function initializeMasterKey(): Promise<boolean> {
  // Already cached — no work to do
  if (cachedMasterKey) return true;

  // Get the access token to derive the master key
  const { getAccessToken } = await import('./auth');
  const token = getAccessToken();
  if (!token) return false;

  // Derive master key from the access token using HKDF-SHA256
  // This is deterministic — same token always produces the same key
  const encoder = new TextEncoder();
  const tokenKeyMaterial = await crypto.subtle.importKey(
    'raw',
    encoder.encode(token),
    { name: 'HKDF' },
    false,
    ['deriveBits'],
  );

  const MASTER_KEY_SALT = encoder.encode('nomados-master-key-salt-v1');

  const derivedKey = await crypto.subtle.deriveBits(
    {
      name: 'HKDF',
      hash: 'SHA-256',
      salt: MASTER_KEY_SALT,
      info: encoder.encode('nomados-master-key'),
    },
    tokenKeyMaterial,
    256, // 32 bytes
  );

  setCachedMasterKey(derivedKey);
  return true;
}