use ed25519_dalek::SigningKey;
use rand::rngs::OsRng;
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;

/// Represents an Ed25519 device keypair.
/// The secret key is stored as bytes; the public key is derived from it.
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct DeviceKeypair {
    /// Base64-encoded secret key bytes (32 bytes seed)
    secret_key_b64: String,
    /// Base64-encoded public key bytes (32 bytes)
    public_key_b64: String,
}

#[allow(dead_code)]
impl DeviceKeypair {
    /// Decode the secret key from base64.
    fn secret_key_bytes(&self) -> Result<Vec<u8>, String> {
        use base64::engine::general_purpose::STANDARD;
        use base64::Engine;
        STANDARD.decode(&self.secret_key_b64).map_err(|e| format!("base64 decode error: {}", e))
    }

    /// Decode the public key from base64.
    fn public_key_bytes(&self) -> Result<Vec<u8>, String> {
        use base64::engine::general_purpose::STANDARD;
        use base64::Engine;
        STANDARD.decode(&self.public_key_b64).map_err(|e| format!("base64 decode error: {}", e))
    }

    /// Get the signing key from stored bytes.
    fn to_signing_key(&self) -> Result<SigningKey, String> {
        let bytes = self.secret_key_bytes()?;
        let secret: [u8; 32] = bytes.as_slice().try_into()
            .map_err(|_| "invalid secret key length".to_string())?;
        Ok(SigningKey::from_bytes(&secret))
    }
}

/// Stores a device keypair to disk (Phase 1: file-based storage).
fn keypair_path() -> Result<PathBuf, String> {
    let dir = dirs::data_dir()
        .ok_or_else(|| "cannot determine data directory".to_string())?;
    let dir = dir.join("nomados");
    fs::create_dir_all(&dir).map_err(|e| format!("cannot create data dir: {}", e))?;
    Ok(dir.join("device_keypair.json"))
}

/// Session token file path.
fn session_path() -> Result<PathBuf, String> {
    let dir = dirs::data_dir()
        .ok_or_else(|| "cannot determine data directory".to_string())?;
    let dir = dir.join("nomados");
    fs::create_dir_all(&dir).map_err(|e| format!("cannot create data dir: {}", e))?;
    Ok(dir.join("session.json"))
}

/// Gateway base URL (Phase 1: hardcoded, later from config).
const GATEWAY_URL: &str = "http://localhost:8080";

#[derive(Serialize, Deserialize, Debug, Clone)]
struct SessionData {
    token: String,
    username: String,
}

/// Generate a new Ed25519 device keypair and store it locally.
#[tauri::command]
pub async fn generate_device_keypair() -> Result<DeviceKeypair, String> {
    let signing_key = SigningKey::generate(&mut OsRng);
    let verifying_key = signing_key.verifying_key();

    let secret_bytes = signing_key.to_bytes();
    let public_bytes = verifying_key.to_bytes();

    use base64::engine::general_purpose::STANDARD;
    use base64::Engine;

    let keypair = DeviceKeypair {
        secret_key_b64: STANDARD.encode(secret_bytes),
        public_key_b64: STANDARD.encode(public_bytes),
    };

    // Store to disk (Phase 1: file-based)
    let path = keypair_path()?;
    let json = serde_json::to_string_pretty(&keypair)
        .map_err(|e| format!("serialize error: {}", e))?;
    fs::write(&path, json).map_err(|e| format!("write error: {}", e))?;

    Ok(keypair)
}

/// Register a new device with the gateway.
#[tauri::command]
pub async fn register(username: String) -> Result<String, String> {
    let path = keypair_path()?;
    if !path.exists() {
        return Err("no device keypair found; generate one first".to_string());
    }

    let json = fs::read_to_string(&path).map_err(|e| format!("read error: {}", e))?;
    let keypair: DeviceKeypair = serde_json::from_str(&json)
        .map_err(|e| format!("parse error: {}", e))?;
    let public_key_b64 = keypair.public_key_b64;

    let client = reqwest::Client::new();
    let body = serde_json::json!({
        "username": username,
        "device_public_key": public_key_b64,
    });

    let resp = client.post(format!("{}/v1/auth/register", GATEWAY_URL))
        .json(&body)
        .send()
        .await
        .map_err(|e| format!("request error: {}", e))?;

    if resp.status().is_success() {
        Ok("registered".to_string())
    } else {
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        Err(format!("registration failed ({}): {}", status, text))
    }
}

/// Initiate login with the gateway. Returns a WebAuthn challenge.
#[tauri::command]
pub async fn login(username: String) -> Result<String, String> {
    let client = reqwest::Client::new();
    let body = serde_json::json!({
        "username": username,
    });

    let resp = client.post(format!("{}/v1/auth/login", GATEWAY_URL))
        .json(&body)
        .send()
        .await
        .map_err(|e| format!("request error: {}", e))?;

    if resp.status().is_success() {
        let challenge = resp.text().await.map_err(|e| format!("read body: {}", e))?;
        Ok(challenge)
    } else {
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        Err(format!("login failed ({}): {}", status, text))
    }
}

/// Verify login with a WebAuthn credential response.
#[tauri::command]
pub async fn verify_login(credential_response: String) -> Result<String, String> {
    let client = reqwest::Client::new();
    let body = serde_json::json!({
        "credential_response": credential_response,
    });

    let resp = client.post(format!("{}/v1/auth/login_verify", GATEWAY_URL))
        .json(&body)
        .send()
        .await
        .map_err(|e| format!("request error: {}", e))?;

    if resp.status().is_success() {
        let resp_body: serde_json::Value = resp.json().await
            .map_err(|e| format!("parse response: {}", e))?;
        let token = resp_body["token"].as_str().unwrap_or("").to_string();
        let username = resp_body["username"].as_str().unwrap_or("").to_string();

        // Store session
        let session = SessionData { token: token.clone(), username };
        let session_json = serde_json::to_string_pretty(&session)
            .map_err(|e| format!("serialize error: {}", e))?;
        let path = session_path()?;
        fs::write(&path, session_json).map_err(|e| format!("write error: {}", e))?;

        Ok(token)
    } else {
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        Err(format!("verify failed ({}): {}", status, text))
    }
}

/// Retrieve the stored session token.
#[tauri::command]
pub async fn get_session_token() -> Result<String, String> {
    let path = session_path()?;
    if !path.exists() {
        return Err("no active session".to_string());
    }

    let json = fs::read_to_string(&path).map_err(|e| format!("read error: {}", e))?;
    let session: SessionData = serde_json::from_str(&json)
        .map_err(|e| format!("parse error: {}", e))?;

    Ok(session.token)
}

/// Logout: clear the stored session.
#[tauri::command]
pub async fn logout() -> Result<String, String> {
    let path = session_path()?;
    if path.exists() {
        fs::remove_file(&path).map_err(|e| format!("delete error: {}", e))?;
    }
    Ok("logged out".to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use ed25519_dalek::{Signer, Verifier};

    #[test]
    fn test_keypair_generation_and_serialization() {
        let signing_key = SigningKey::generate(&mut OsRng);
        let verifying_key = signing_key.verifying_key();

        let secret_bytes = signing_key.to_bytes();
        let public_bytes = verifying_key.to_bytes();

        use base64::engine::general_purpose::STANDARD;
        use base64::Engine;

        let keypair = DeviceKeypair {
            secret_key_b64: STANDARD.encode(secret_bytes),
            public_key_b64: STANDARD.encode(public_bytes),
        };

        // Round-trip through JSON
        let json = serde_json::to_string(&keypair).unwrap();
        let decoded: DeviceKeypair = serde_json::from_str(&json).unwrap();

        assert_eq!(keypair.secret_key_b64, decoded.secret_key_b64);
        assert_eq!(keypair.public_key_b64, decoded.public_key_b64);
    }

    #[test]
    fn test_keypair_signing_roundtrip() {
        let signing_key = SigningKey::generate(&mut OsRng);
        let verifying_key = signing_key.verifying_key();

        let message = b"test message for nomados auth";
        let signature = signing_key.sign(message);

        // Verify the signature
        assert!(verifying_key.verify(message, &signature).is_ok());
    }

    #[test]
    fn test_session_data_serialization() {
        let session = SessionData {
            token: "test-token-abc123".to_string(),
            username: "testuser".to_string(),
        };

        let json = serde_json::to_string(&session).unwrap();
        let decoded: SessionData = serde_json::from_str(&json).unwrap();

        assert_eq!(session.token, decoded.token);
        assert_eq!(session.username, decoded.username);
    }
}