use argon2::{Argon2, Params, Algorithm, Version};
use hkdf::Hkdf;
use sha2::Sha256;

const ARGON2_MEMORY: u32 = 65536; // 64MB
const ARGON2_ITERATIONS: u32 = 3;
const ARGON2_PARALLELISM: u32 = 4;

pub fn derive_master_key(password: &[u8], salt: &[u8]) -> Result<[u8; 32], String> {
    let params = Params::new(ARGON2_MEMORY, ARGON2_ITERATIONS, ARGON2_PARALLELISM, Some(32))
        .map_err(|e| format!("argon2 params error: {}", e))?;
    let argon2 = Argon2::new(Algorithm::Argon2id, Version::V0x13, params);
    let mut key = [0u8; 32];
    argon2
        .hash_password_into(password, salt, &mut key)
        .map_err(|e| format!("argon2 hash error: {}", e))?;
    Ok(key)
}

pub fn derive_workspace_key(master_key: &[u8; 32], workspace_id: &[u8]) -> Result<[u8; 32], String> {
    let hkdf = Hkdf::<Sha256>::new(Some(b"nomados-workspace-key"), master_key);
    let mut key = [0u8; 32];
    hkdf.expand(workspace_id, &mut key)
        .map_err(|e| format!("hkdf expand error: {}", e))?;
    Ok(key)
}

pub fn derive_file_key(workspace_key: &[u8; 32], file_id: &[u8]) -> Result<[u8; 32], String> {
    let hkdf = Hkdf::<Sha256>::new(Some(b"nomados-file-key"), workspace_key);
    let mut key = [0u8; 32];
    hkdf.expand(file_id, &mut key)
        .map_err(|e| format!("hkdf expand error: {}", e))?;
    Ok(key)
}