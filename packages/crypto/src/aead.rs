use chacha20poly1305::{ChaCha20Poly1305, Key, Nonce, aead::Aead, aead::OsRng, AeadCore, KeyInit};

const NONCE_SIZE: usize = 12; // ChaCha20Poly1305 uses 12-byte nonces

pub fn encrypt(key: &[u8; 32], plaintext: &[u8]) -> Result<Vec<u8>, String> {
    let cipher = ChaCha20Poly1305::new(Key::from_slice(key));
    let nonce = ChaCha20Poly1305::generate_nonce(&mut OsRng);
    let ciphertext = cipher
        .encrypt(&nonce, plaintext)
        .map_err(|e| format!("encryption error: {}", e))?;
    let mut result = Vec::with_capacity(NONCE_SIZE + ciphertext.len());
    result.extend_from_slice(&nonce);
    result.extend_from_slice(&ciphertext);
    Ok(result)
}

pub fn decrypt(key: &[u8; 32], data: &[u8]) -> Result<Vec<u8>, String> {
    if data.len() < NONCE_SIZE {
        return Err("data too short: missing nonce".into());
    }
    let (nonce_bytes, ciphertext) = data.split_at(NONCE_SIZE);
    let nonce = Nonce::from_slice(nonce_bytes);
    let cipher = ChaCha20Poly1305::new(Key::from_slice(key));
    cipher
        .decrypt(nonce, ciphertext)
        .map_err(|e| format!("decryption error: {}", e))
}

pub fn encrypt_stream(key: &[u8; 32], chunks: &[&[u8]]) -> Result<Vec<Vec<u8>>, String> {
    chunks.iter().map(|chunk| encrypt(key, chunk)).collect()
}

pub fn decrypt_stream(key: &[u8; 32], encrypted_chunks: &[Vec<u8>]) -> Result<Vec<Vec<u8>>, String> {
    encrypted_chunks.iter().map(|chunk| decrypt(key, chunk)).collect()
}