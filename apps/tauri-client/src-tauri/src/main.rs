#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod auth;
mod crypto;
mod stream;
mod workspace;

fn main() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            // Auth commands
            auth::generate_device_keypair,
            auth::register,
            auth::login,
            auth::verify_login,
            auth::get_session_token,
            auth::logout,
            // Crypto commands
            crypto::encrypt_local,
            crypto::decrypt_local,
            crypto::derive_keys,
            crypto::derive_workspace_key,
            crypto::derive_file_key,
            // Workspace commands
            workspace::list_workspaces,
            workspace::create_workspace,
            workspace::pause_workspace,
            workspace::resume_workspace,
            workspace::destroy_workspace,
            // Stream commands
            stream::connect_stream,
            stream::disconnect_stream,
            stream::send_input,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}