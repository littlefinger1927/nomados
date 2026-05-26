use nomados_crypto::{self as nc, zeroize::zeroize_buffer};

/// Encrypt data with a key. Returns nonce (12 bytes) + ciphertext.
#[tauri::command]
pub async fn encrypt_local(data: Vec<u8>, key: Vec<u8>) -> Result<Vec<u8>, String> {
    if key.len() != 32 {
        return Err("key must be 32 bytes".to_string());
    }
    let key_arr: [u8; 32] = key.as_slice().try_into()
        .map_err(|_| "invalid key length".to_string())?;
    let result = nc::encrypt(&key_arr, &data)?;
    Ok(result)
}

/// Decrypt data that was encrypted with encrypt_local.
#[tauri::command]
pub async fn decrypt_local(encrypted: Vec<u8>, key: Vec<u8>) -> Result<Vec<u8>, String> {
    if key.len() != 32 {
        return Err("key must be 32 bytes".to_string());
    }
    let key_arr: [u8; 32] = key.as_slice().try_into()
        .map_err(|_| "invalid key length".to_string())?;
    let mut key_material = key_arr;
    let result = nc::decrypt(&key_arr, &encrypted)?;
    zeroize_buffer(&mut key_material);
    Ok(result)
}

/// Derive a master key from password and salt using Argon2id.
#[tauri::command]
pub async fn derive_keys(password: String, salt: String) -> Result<Vec<u8>, String> {
    let key = nc::derive_master_key(password.as_bytes(), salt.as_bytes())?;
    Ok(key.to_vec())
}

/// Derive a workspace key from a master key and workspace ID.
#[tauri::command]
pub async fn derive_workspace_key(master_key: Vec<u8>, workspace_id: String) -> Result<Vec<u8>, String> {
    if master_key.len() != 32 {
        return Err("master_key must be 32 bytes".to_string());
    }
    let key_arr: [u8; 32] = master_key.as_slice().try_into()
        .map_err(|_| "invalid master key length".to_string())?;
    let ws_key = nc::derive_workspace_key(&key_arr, workspace_id.as_bytes())?;
    Ok(ws_key.to_vec())
}

/// Derive a file key from a workspace key and file ID.
#[tauri::command]
pub async fn derive_file_key(workspace_key: Vec<u8>, file_id: String) -> Result<Vec<u8>, String> {
    if workspace_key.len() != 32 {
        return Err("workspace_key must be 32 bytes".to_string());
    }
    let key_arr: [u8; 32] = workspace_key.as_slice().try_into()
        .map_err(|_| "invalid workspace key length".to_string())?;
    let file_key = nc::derive_file_key(&key_arr, file_id.as_bytes())?;
    Ok(file_key.to_vec())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_encrypt_decrypt_roundtrip() {
        let data = b"hello nomados world";
        let key: [u8; 32] = [42u8; 32];

        let encrypted = nc::encrypt(&key, data).unwrap();
        let decrypted = nc::decrypt(&key, &encrypted).unwrap();

        assert_eq!(data.to_vec(), decrypted);
    }

    #[test]
    fn test_encrypt_decrypt_different_data_sizes() {
        for size in [0, 1, 15, 16, 100, 1024] {
            let data = vec![0xABu8; size];
            let key: [u8; 32] = [7u8; 32];

            let encrypted = nc::encrypt(&key, &data).unwrap();
            let decrypted = nc::decrypt(&key, &encrypted).unwrap();

            assert_eq!(data, decrypted, "roundtrip failed for size {}", size);
        }
    }

    #[test]
    fn test_derive_master_key() {
        let key1 = nc::derive_master_key(b"password", b"salt12345").unwrap();
        let key2 = nc::derive_master_key(b"password", b"salt12345").unwrap();
        assert_eq!(key1, key2, "same input should produce same key");

        let key3 = nc::derive_master_key(b"password", b"different-salt").unwrap();
        assert_ne!(key1, key3, "different salt should produce different key");
    }

    #[test]
    fn test_derive_workspace_key() {
        let master_key = nc::derive_master_key(b"password", b"salt12345").unwrap();
        let ws1 = nc::derive_workspace_key(&master_key, b"workspace-1").unwrap();
        let ws2 = nc::derive_workspace_key(&master_key, b"workspace-2").unwrap();
        assert_ne!(ws1, ws2, "different workspace IDs should produce different keys");

        let ws1_again = nc::derive_workspace_key(&master_key, b"workspace-1").unwrap();
        assert_eq!(ws1, ws1_again, "same input should produce same key");
    }

    #[test]
    fn test_derive_file_key() {
        let master_key = nc::derive_master_key(b"password", b"salt12345").unwrap();
        let ws_key = nc::derive_workspace_key(&master_key, b"ws-1").unwrap();
        let f1 = nc::derive_file_key(&ws_key, b"file-1").unwrap();
        let f2 = nc::derive_file_key(&ws_key, b"file-2").unwrap();
        assert_ne!(f1, f2, "different file IDs should produce different keys");

        let f1_again = nc::derive_file_key(&ws_key, b"file-1").unwrap();
        assert_eq!(f1, f1_again, "same input should produce same key");
    }

    #[test]
    fn test_encrypt_with_derived_key() {
        let master_key = nc::derive_master_key(b"passphrase", b"random-salt").unwrap();
        let ws_key = nc::derive_workspace_key(&master_key, b"my-workspace").unwrap();
        let file_key = nc::derive_file_key(&ws_key, b"my-file").unwrap();

        let plaintext = b"important secret data";
        let encrypted = nc::encrypt(&file_key, plaintext).unwrap();
        let decrypted = nc::decrypt(&file_key, &encrypted).unwrap();

        assert_eq!(plaintext.to_vec(), decrypted);
    }
}