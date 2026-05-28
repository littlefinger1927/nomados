/**
 * AES-256-GCM encryption/decryption using the Web Crypto API.
 *
 * Binary format for encrypted output:
 *   [IV_LENGTH (2 bytes, big-endian)] [IV] [ciphertext + auth tag]
 *
 * The IV is 12 bytes (standard for AES-GCM). The auth tag is appended
 * to the ciphertext by Web Crypto's encrypt() call (16 bytes for AES-GCM).
 */

/**
 * Encrypt data using AES-256-GCM.
 *
 * @param data - Plaintext data to encrypt
 * @param key - 32-byte AES key (ArrayBuffer)
 * @returns Encrypted payload: [2-byte IV length (big-endian)] [IV] [ciphertext+tag]
 */
export async function encrypt(
  data: ArrayBuffer,
  key: ArrayBuffer,
): Promise<ArrayBuffer> {
  // Generate a random 12-byte IV for AES-GCM
  const iv = crypto.getRandomValues(new Uint8Array(12));

  // Import the raw key for use with AES-GCM
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    key,
    { name: 'AES-GCM' },
    false,
    ['encrypt'],
  );

  // Encrypt — Web Crypto returns ciphertext + 16-byte auth tag concatenated
  const encrypted = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv },
    cryptoKey,
    data,
  );

  // Build output: [2-byte IV length (big-endian)] [IV] [ciphertext+tag]
  const result = new Uint8Array(2 + iv.length + encrypted.byteLength);
  const view = new DataView(result.buffer);
  view.setUint16(0, iv.length, false); // big-endian
  result.set(iv, 2);
  result.set(new Uint8Array(encrypted), 2 + iv.length);

  return result.buffer;
}

/**
 * Decrypt data using AES-256-GCM.
 *
 * @param data - Encrypted payload: [2-byte IV length] [IV] [ciphertext+tag]
 * @param key - 32-byte AES key (ArrayBuffer)
 * @returns Decrypted plaintext (ArrayBuffer)
 */
export async function decrypt(
  data: ArrayBuffer,
  key: ArrayBuffer,
): Promise<ArrayBuffer> {
  const view = new DataView(data);
  const ivLength = view.getUint16(0, false); // big-endian
  const iv = new Uint8Array(data, 2, ivLength);
  const ciphertext = data.slice(2 + ivLength);

  // Import the raw key for use with AES-GCM
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    key,
    { name: 'AES-GCM' },
    false,
    ['decrypt'],
  );

  // Decrypt — Web Crypto expects ciphertext + auth tag concatenated
  return crypto.subtle.decrypt({ name: 'AES-GCM', iv }, cryptoKey, ciphertext);
}