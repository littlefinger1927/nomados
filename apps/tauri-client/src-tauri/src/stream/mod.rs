use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Mutex;

/// Input event types for remote desktop streaming.
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(tag = "event_type")]
pub enum InputEvent {
    #[serde(rename = "mouse_move")]
    MouseMove { x: i32, y: i32 },
    #[serde(rename = "mouse_click")]
    MouseClick { x: i32, y: i32, button: String },
    #[serde(rename = "key_press")]
    KeyPress { key: String },
    #[serde(rename = "key_release")]
    KeyRelease { key: String },
}

/// Manages the state of WebRTC stream connections.
/// Phase 1: Stub implementation; actual pion/webrtc integration will come later.
pub struct StreamManager {
    connections: Mutex<HashMap<String, bool>>,
}

impl StreamManager {
    fn new() -> Self {
        Self {
            connections: Mutex::new(HashMap::new()),
        }
    }

    fn is_connected(&self, workspace_id: &str) -> bool {
        let connections = self.connections.lock().unwrap();
        connections.get(workspace_id).copied().unwrap_or(false)
    }

    fn connect(&self, workspace_id: &str) {
        let mut connections = self.connections.lock().unwrap();
        connections.insert(workspace_id.to_string(), true);
    }

    fn disconnect(&self, workspace_id: &str) {
        let mut connections = self.connections.lock().unwrap();
        connections.remove(workspace_id);
    }
}

/// Global stream manager instance.
static STREAM_MANAGER: std::sync::OnceLock<StreamManager> = std::sync::OnceLock::new();

fn get_stream_manager() -> &'static StreamManager {
    STREAM_MANAGER.get_or_init(StreamManager::new)
}

/// Establish a WebRTC stream connection to a workspace.
/// Phase 1: Stub — marks the workspace as connected without actual WebRTC.
#[tauri::command]
pub async fn connect_stream(workspace_id: String) -> Result<String, String> {
    let manager = get_stream_manager();
    if manager.is_connected(&workspace_id) {
        return Err(format!("already connected to workspace {}", workspace_id));
    }
    // Phase 1: Stub — in a real implementation, this would:
    // 1. Fetch WebRTC config from the gateway
    // 2. Create a RTCPeerConnection
    // 3. Set up data channels for input events
    // 4. Establish media stream for video/audio
    manager.connect(&workspace_id);
    Ok(format!("connected to workspace {}", workspace_id))
}

/// Close the WebRTC stream connection for a workspace.
/// Phase 1: Stub — removes the workspace from the connection map.
#[tauri::command]
pub async fn disconnect_stream(workspace_id: String) -> Result<String, String> {
    let manager = get_stream_manager();
    if !manager.is_connected(&workspace_id) {
        return Err(format!("not connected to workspace {}", workspace_id));
    }
    manager.disconnect(&workspace_id);
    Ok(format!("disconnected from workspace {}", workspace_id))
}

/// Send an input event (mouse/keyboard) to a workspace stream.
/// Phase 1: Stub — serializes the event but does not send it anywhere.
#[tauri::command]
pub async fn send_input(workspace_id: String, event: InputEvent) -> Result<String, String> {
    let manager = get_stream_manager();
    if !manager.is_connected(&workspace_id) {
        return Err(format!("not connected to workspace {}", workspace_id));
    }

    // Phase 1: Serialize and log the event (in production this would send over data channel)
    let _event_json = serde_json::to_string(&event)
        .map_err(|e| format!("serialize error: {}", e))?;

    // In the real implementation, this would:
    // 1. Serialize the InputEvent
    // 2. Send it over the WebRTC data channel
    Ok("input sent".to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_input_event_serialization_mouse_move() {
        let event = InputEvent::MouseMove { x: 100, y: 200 };
        let json = serde_json::to_string(&event).unwrap();
        let decoded: InputEvent = serde_json::from_str(&json).unwrap();

        // Verify round-trip
        let json2 = serde_json::to_string(&decoded).unwrap();
        assert_eq!(json, json2);
    }

    #[test]
    fn test_input_event_serialization_mouse_click() {
        let event = InputEvent::MouseClick {
            x: 50,
            y: 75,
            button: "left".to_string(),
        };
        let json = serde_json::to_string(&event).unwrap();
        assert!(json.contains("mouse_click"));
        assert!(json.contains("left"));

        let decoded: InputEvent = serde_json::from_str(&json).unwrap();
        let json2 = serde_json::to_string(&decoded).unwrap();
        assert_eq!(json, json2);
    }

    #[test]
    fn test_input_event_serialization_key_press() {
        let event = InputEvent::KeyPress {
            key: "Enter".to_string(),
        };
        let json = serde_json::to_string(&event).unwrap();
        assert!(json.contains("key_press"));

        let decoded: InputEvent = serde_json::from_str(&json).unwrap();
        let json2 = serde_json::to_string(&decoded).unwrap();
        assert_eq!(json, json2);
    }

    #[test]
    fn test_input_event_key_release() {
        let event = InputEvent::KeyRelease {
            key: "Shift".to_string(),
        };
        let json = serde_json::to_string(&event).unwrap();
        assert!(json.contains("key_release"));

        let decoded: InputEvent = serde_json::from_str(&json).unwrap();
        let json2 = serde_json::to_string(&decoded).unwrap();
        assert_eq!(json, json2);
    }

    #[test]
    fn test_stream_manager_connect_disconnect() {
        let manager = StreamManager::new();

        assert!(!manager.is_connected("ws-1"));

        manager.connect("ws-1");
        assert!(manager.is_connected("ws-1"));
        assert!(!manager.is_connected("ws-2"));

        manager.disconnect("ws-1");
        assert!(!manager.is_connected("ws-1"));
    }

    #[test]
    fn test_stream_manager_double_connect() {
        let manager = StreamManager::new();
        manager.connect("ws-1");
        manager.connect("ws-1"); // should overwrite, still true
        assert!(manager.is_connected("ws-1"));
    }
}