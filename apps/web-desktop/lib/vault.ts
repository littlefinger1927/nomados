/**
 * Vault service client for NomadOS web desktop.
 *
 * Provides functions for key derivation via the vault gRPC gateway.
 * The gateway serializes proto `bytes` fields as base64 in JSON,
 * so we handle base64 encoding/decoding for master_key and encrypted_key.
 */

import { apiPost } from './auth';

/**
 * Convert an ArrayBuffer to a base64 string.
 * Used to encode master_key bytes for JSON transport.
 */
function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

/**
 * Convert a base64 string to an ArrayBuffer.
 * Used to decode encrypted_key bytes from JSON responses.
 */
function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

/**
 * Derive a workspace encryption key from a master key.
 *
 * The client must first derive the master key locally using PBKDF2
 * (via deriveMasterKey in auth.ts), then pass it here along with
 * the workspace ID. The vault service derives the workspace key
 * server-side using HKDF-SHA256 and returns the encrypted key.
 *
 * @param masterKey - 32-byte master key derived from user passphrase
 * @param workspaceId - UUID of the workspace
 * @returns Derived workspace key as ArrayBuffer
 */
export async function deriveWorkspaceKey(
  masterKey: ArrayBuffer,
  workspaceId: string,
): Promise<ArrayBuffer> {
  const response = await apiPost('/v1/vault/derive_workspace_key', {
    master_key: arrayBufferToBase64(masterKey),
    workspace_id: { value: workspaceId },
  });

  if (!response.ok) {
    const errorBody = await response.text();
    throw new Error(`deriveWorkspaceKey failed (${response.status}): ${errorBody}`);
  }

  const data = await response.json();
  return base64ToArrayBuffer(data.encrypted_key);
}

/**
 * Derive a file encryption key from a master key.
 *
 * The client must provide the master key. The vault service will
 * first derive the workspace key, then derive the file key from it.
 *
 * @param masterKey - 32-byte master key derived from user passphrase
 * @param workspaceId - UUID of the workspace
 * @param fileId - UUID of the file
 * @returns Derived file key as ArrayBuffer
 */
export async function deriveFileKey(
  masterKey: ArrayBuffer,
  workspaceId: string,
  fileId: string,
): Promise<ArrayBuffer> {
  const response = await apiPost('/v1/vault/derive_file_key', {
    master_key: arrayBufferToBase64(masterKey),
    workspace_id: { value: workspaceId },
    file_id: { value: fileId },
  });

  if (!response.ok) {
    const errorBody = await response.text();
    throw new Error(`deriveFileKey failed (${response.status}): ${errorBody}`);
  }

  const data = await response.json();
  return base64ToArrayBuffer(data.encrypted_key);
}

/**
 * Rotate a workspace encryption key.
 *
 * The client must provide the master key. The vault service will
 * derive a new workspace key and publish a rotation event.
 *
 * @param masterKey - 32-byte master key derived from user passphrase
 * @param workspaceId - UUID of the workspace
 */
export async function rotateWorkspaceKey(
  masterKey: ArrayBuffer,
  workspaceId: string,
): Promise<void> {
  const response = await apiPost('/v1/vault/rotate_workspace_key', {
    master_key: arrayBufferToBase64(masterKey),
    workspace_id: { value: workspaceId },
  });

  if (!response.ok) {
    const errorBody = await response.text();
    throw new Error(`rotateWorkspaceKey failed (${response.status}): ${errorBody}`);
  }
}