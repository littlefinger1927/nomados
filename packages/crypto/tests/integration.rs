use nomados_crypto::{derive_master_key, derive_workspace_key, derive_file_key};

#[test]
fn test_master_key_derivation() {
    let password = b"test-password-123";
    let salt = b"nomados-user-salt";
    let key = derive_master_key(password, salt).unwrap();
    assert_eq!(key.len(), 32);
}

#[test]
fn test_workspace_key_derivation() {
    let master_key = [0u8; 32];
    let workspace_id = b"workspace-uuid-123";
    let key = derive_workspace_key(&master_key, workspace_id).unwrap();
    assert_eq!(key.len(), 32);
}

#[test]
fn test_file_key_derivation() {
    let workspace_key = [0u8; 32];
    let file_id = b"file-uuid-456";
    let key = derive_file_key(&workspace_key, file_id).unwrap();
    assert_eq!(key.len(), 32);
}

#[test]
fn test_different_salts_produce_different_keys() {
    let password = b"same-password";
    let key1 = derive_master_key(password, b"salt-001").unwrap();
    let key2 = derive_master_key(password, b"salt-002").unwrap();
    assert_ne!(key1, key2);
}