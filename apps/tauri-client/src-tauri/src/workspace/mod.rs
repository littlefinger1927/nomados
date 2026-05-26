use serde::{Deserialize, Serialize};

/// Workspace lifecycle states.
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum WorkspaceState {
    Creating,
    Running,
    Paused,
    Stopping,
    Stopped,
}

/// A workspace resource.
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Workspace {
    pub id: String,
    pub name: String,
    pub state: WorkspaceState,
    pub created_at: String,
}

/// Gateway base URL (Phase 1: hardcoded, later from config).
const GATEWAY_URL: &str = "http://localhost:8080";

/// Session token file path (shared with auth module).
fn session_path() -> Result<std::path::PathBuf, String> {
    let dir = dirs::data_dir()
        .ok_or_else(|| "cannot determine data directory".to_string())?;
    let dir = dir.join("nomados");
    std::fs::create_dir_all(&dir).map_err(|e| format!("cannot create data dir: {}", e))?;
    Ok(dir.join("session.json"))
}

#[derive(Serialize, Deserialize, Debug, Clone)]
struct SessionData {
    token: String,
    username: String,
}

/// Get the stored session token for authorization headers.
fn get_auth_token() -> Result<String, String> {
    let path = session_path()?;
    if !path.exists() {
        return Err("no active session; please log in first".to_string());
    }
    let json = std::fs::read_to_string(&path).map_err(|e| format!("read error: {}", e))?;
    let session: SessionData = serde_json::from_str(&json)
        .map_err(|e| format!("parse error: {}", e))?;
    Ok(session.token)
}

/// List all workspaces for the current user.
#[tauri::command]
pub async fn list_workspaces() -> Result<Vec<Workspace>, String> {
    let token = get_auth_token()?;
    let client = reqwest::Client::new();

    let resp = client.get(format!("{}/v1/workspace/list", GATEWAY_URL))
        .header("Authorization", format!("Bearer {}", token))
        .send()
        .await
        .map_err(|e| format!("request error: {}", e))?;

    if resp.status().is_success() {
        let workspaces: Vec<Workspace> = resp.json().await
            .map_err(|e| format!("parse response: {}", e))?;
        Ok(workspaces)
    } else {
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        Err(format!("list workspaces failed ({}): {}", status, text))
    }
}

/// Create a new workspace.
#[tauri::command]
pub async fn create_workspace(name: String) -> Result<Workspace, String> {
    let token = get_auth_token()?;
    let client = reqwest::Client::new();

    let body = serde_json::json!({
        "name": name,
    });

    let resp = client.post(format!("{}/v1/workspace/create", GATEWAY_URL))
        .header("Authorization", format!("Bearer {}", token))
        .json(&body)
        .send()
        .await
        .map_err(|e| format!("request error: {}", e))?;

    if resp.status().is_success() {
        let workspace: Workspace = resp.json().await
            .map_err(|e| format!("parse response: {}", e))?;
        Ok(workspace)
    } else {
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        Err(format!("create workspace failed ({}): {}", status, text))
    }
}

/// Pause a running workspace.
#[tauri::command]
pub async fn pause_workspace(id: String) -> Result<Workspace, String> {
    let token = get_auth_token()?;
    let client = reqwest::Client::new();

    let body = serde_json::json!({
        "id": id,
    });

    let resp = client.post(format!("{}/v1/workspace/pause", GATEWAY_URL))
        .header("Authorization", format!("Bearer {}", token))
        .json(&body)
        .send()
        .await
        .map_err(|e| format!("request error: {}", e))?;

    if resp.status().is_success() {
        let workspace: Workspace = resp.json().await
            .map_err(|e| format!("parse response: {}", e))?;
        Ok(workspace)
    } else {
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        Err(format!("pause workspace failed ({}): {}", status, text))
    }
}

/// Resume a paused workspace.
#[tauri::command]
pub async fn resume_workspace(id: String) -> Result<Workspace, String> {
    let token = get_auth_token()?;
    let client = reqwest::Client::new();

    let body = serde_json::json!({
        "id": id,
    });

    let resp = client.post(format!("{}/v1/workspace/resume", GATEWAY_URL))
        .header("Authorization", format!("Bearer {}", token))
        .json(&body)
        .send()
        .await
        .map_err(|e| format!("request error: {}", e))?;

    if resp.status().is_success() {
        let workspace: Workspace = resp.json().await
            .map_err(|e| format!("parse response: {}", e))?;
        Ok(workspace)
    } else {
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        Err(format!("resume workspace failed ({}): {}", status, text))
    }
}

/// Destroy a workspace permanently.
#[tauri::command]
pub async fn destroy_workspace(id: String) -> Result<String, String> {
    let token = get_auth_token()?;
    let client = reqwest::Client::new();

    let body = serde_json::json!({
        "id": id,
    });

    let resp = client.post(format!("{}/v1/workspace/destroy", GATEWAY_URL))
        .header("Authorization", format!("Bearer {}", token))
        .json(&body)
        .send()
        .await
        .map_err(|e| format!("request error: {}", e))?;

    if resp.status().is_success() {
        Ok("destroyed".to_string())
    } else {
        let status = resp.status();
        let text = resp.text().await.unwrap_or_default();
        Err(format!("destroy workspace failed ({}): {}", status, text))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_workspace_state_serialization() {
        let states = vec![
            WorkspaceState::Creating,
            WorkspaceState::Running,
            WorkspaceState::Paused,
            WorkspaceState::Stopping,
            WorkspaceState::Stopped,
        ];

        for state in &states {
            let json = serde_json::to_string(state).unwrap();
            let decoded: WorkspaceState = serde_json::from_str(&json).unwrap();
            assert_eq!(state, &decoded);
        }
    }

    #[test]
    fn test_workspace_state_json_values() {
        assert_eq!(
            serde_json::to_string(&WorkspaceState::Creating).unwrap(),
            "\"creating\""
        );
        assert_eq!(
            serde_json::to_string(&WorkspaceState::Running).unwrap(),
            "\"running\""
        );
        assert_eq!(
            serde_json::to_string(&WorkspaceState::Paused).unwrap(),
            "\"paused\""
        );
    }

    #[test]
    fn test_workspace_serialization() {
        let ws = Workspace {
            id: "ws-123".to_string(),
            name: "my-workspace".to_string(),
            state: WorkspaceState::Running,
            created_at: "2025-01-01T00:00:00Z".to_string(),
        };

        let json = serde_json::to_string(&ws).unwrap();
        let decoded: Workspace = serde_json::from_str(&json).unwrap();

        assert_eq!(ws.id, decoded.id);
        assert_eq!(ws.name, decoded.name);
        assert_eq!(ws.state, decoded.state);
        assert_eq!(ws.created_at, decoded.created_at);
    }

    #[test]
    fn test_workspace_state_from_str() {
        let json = "\"running\"";
        let state: WorkspaceState = serde_json::from_str(json).unwrap();
        assert_eq!(state, WorkspaceState::Running);

        let json = "\"paused\"";
        let state: WorkspaceState = serde_json::from_str(json).unwrap();
        assert_eq!(state, WorkspaceState::Paused);
    }
}