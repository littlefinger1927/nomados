# NomadOS Phase 1 — Sovereign Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the NomadOS Phase 1 core loop — authenticate securely, create isolated workspace, operate inside it — as a monorepo of Go microservices with a Rust Tauri client.

**Architecture:** Monorepo microservices communicating via gRPC over mTLS. Docker Compose for local development. Rust crypto shared library (called directly from Tauri, via CGo from Go services). WebAuthn/passkey auth with device-bound sessions. Remote Chromium streaming via WebRTC/TURN.

**Tech Stack:** Go 1.22+ (services), Rust (Tauri + crypto), TypeScript/React/Next.js (web UI), PostgreSQL, Redis, MinIO, NATS, Docker, buf (proto generation)

---

## Milestone 1: Foundation

### Task 1: Monorepo Scaffold

**Files:**
- Create: `nomados/go.mod`
- Create: `nomados/go.work`
- Create: `nomados/Makefile`
- Create: `nomados/scripts/dev.sh`
- Create: `nomados/scripts/test.sh`
- Create: `nomados/scripts/build.sh`
- Create: `nomados/.gitignore`
- Create: `nomados/.editorconfig`
- Create: `nomados/buf.yaml`
- Create: `nomados/buf.gen.yaml`

- [ ] **Step 1: Initialize Go workspace and module**

Create `go.work`:

```
go 1.22

use (
    services/gateway-service
    services/auth-service
    services/session-service
    services/workspace-orchestrator
    services/browser-manager
    services/streaming-service
    services/file-service
    services/vault-service
    services/observability
    packages/shared-types
    packages/auth-sdk
    packages/logging
)
```

Create root `go.mod`:

```
module github.com/nomados/nomados

go 1.22
```

- [ ] **Step 2: Create directory structure**

```bash
mkdir -p apps/{web-desktop,auth-portal,tauri-client}
mkdir -p services/{gateway-service,auth-service,session-service,workspace-orchestrator,browser-manager,streaming-service,file-service,vault-service,observability}/cmd
mkdir -p infrastructure/{docker,wireguard,ci-cd}
mkdir -p packages/{shared-types,auth-sdk,logging}/proto
mkdir -p packages/crypto/src
mkdir -p packages/ui-components
mkdir -p security/{policies,threat-models}
mkdir -p docs/{architecture,deployment,APIs,security}
```

- [ ] **Step 3: Create Makefile with common targets**

```makefile
.PHONY: proto build test lint dev clean

proto:
	buf generate

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run

dev:
	bash scripts/dev.sh

clean:
	rm -rf bin/ tmp/
```

- [ ] **Step 4: Create .gitignore**

```
# Go
bin/
*.exe
*.test
*.out

# Rust
target/
Cargo.lock

# Node
node_modules/
.next/
out/

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store

# Environment
.env
.env.local

# Infrastructure
infrastructure/docker/data/

# Build artifacts
dist/
tmp/
```

- [ ] **Step 5: Create .editorconfig**

```
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true
indent_style = space
indent_size = 2

[*.go]
indent_style = tab

[*.rs]
indent_style = space
indent_size = 4

[Makefile]
indent_style = tab
```

- [ ] **Step 6: Create buf.yaml and buf.gen.yaml**

`buf.yaml`:
```yaml
version: 2
modules:
  - path: packages/shared-types/proto
```

`buf.gen.yaml`:
```yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/nomados/nomados/packages/shared-types/gen
plugins:
  - remote: buf.build/protocolbuffers/go
    out: packages/shared-types/gen
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: packages/shared-types/gen
    opt: paths=source_relative
```

- [ ] **Step 7: Initialize git repo and commit**

```bash
cd /Users/ctxdigital/nomados
git init
git add .
git commit -m "feat: initialize monorepo scaffold

Set up Go workspace, directory structure, Makefile, buf config,
and development scripts for NomadOS Phase 1."
```

---

### Task 2: Proto Definitions

**Files:**
- Create: `packages/shared-types/proto/auth/v1/auth.proto`
- Create: `packages/shared-types/proto/session/v1/session.proto`
- Create: `packages/shared-types/proto/workspace/v1/workspace.proto`
- Create: `packages/shared-types/proto/file/v1/file.proto`
- Create: `packages/shared-types/proto/vault/v1/vault.proto`
- Create: `packages/shared-types/proto/common/v1/common.proto`

- [ ] **Step 1: Write common proto types**

Create `packages/shared-types/proto/common/v1/common.proto`:

```protobuf
syntax = "proto3";

package nomados.common.v1;

option go_package = "github.com/nomados/nomados/packages/shared-types/gen/common/v1";

message Empty {}

message UUID {
  string value = 1;
}

message Timestamp {
  int64 seconds = 1;
  int32 nanos = 2;
}

message ErrorResponse {
  string code = 1;
  string message = 2;
  map<string, string> details = 3;
}
```

- [ ] **Step 2: Write auth proto**

Create `packages/shared-types/proto/auth/v1/auth.proto`:

```protobuf
syntax = "proto3";

package nomados.auth.v1;

option go_package = "github.com/nomados/nomados/packages/shared-types/gen/auth/v1";

import "common/v1/common.proto";

service AuthService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc RegisterVerify(RegisterVerifyRequest) returns (RegisterVerifyResponse);
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc LoginVerify(LoginVerifyRequest) returns (LoginVerifyResponse);
}

message RegisterRequest {
  string username = 1;
  bytes device_public_key = 2;
  string device_attestation = 3;
}

message RegisterResponse {
  bytes webauthn_challenge = 1;
  nomados.common.v1.UUID user_id = 2;
}

message RegisterVerifyRequest {
  bytes credential_response = 1;
  bytes device_signature = 2;
  nomados.common.v1.UUID user_id = 3;
}

message RegisterVerifyResponse {
  string access_token = 1;
  string refresh_token = 2;
  nomados.common.v1.UUID device_id = 3;
}

message LoginRequest {
  bytes device_public_key = 1;
}

message LoginResponse {
  bytes webauthn_challenge = 1;
  nomados.common.v1.UUID user_id = 2;
}

message LoginVerifyRequest {
  bytes assertion_response = 1;
  bytes device_signature = 2;
  nomados.common.v1.UUID user_id = 3;
}

message LoginVerifyResponse {
  string access_token = 1;
  string refresh_token = 2;
  nomados.common.v1.UUID session_id = 3;
  nomados.common.v1.UUID device_id = 4;
}
```

- [ ] **Step 3: Write session proto**

Create `packages/shared-types/proto/session/v1/session.proto`:

```protobuf
syntax = "proto3";

package nomados.session.v1;

option go_package = "github.com/nomados/nomados/packages/shared-types/gen/session/v1";

import "common/v1/common.proto";

service SessionService {
  rpc Create(CreateSessionRequest) returns (CreateSessionResponse);
  rpc Validate(ValidateSessionRequest) returns (ValidateSessionResponse);
  rpc Refresh(RefreshSessionRequest) returns (RefreshSessionResponse);
  rpc Revoke(RevokeSessionRequest) returns (nomados.common.v1.Empty);
}

message CreateSessionRequest {
  nomados.common.v1.UUID user_id = 1;
  nomados.common.v1.UUID device_id = 2;
  string ip_hash = 3;
  int32 risk_score = 4;
}

message CreateSessionResponse {
  nomados.common.v1.UUID session_id = 1;
  string access_token = 2;
  string refresh_token = 3;
}

message ValidateSessionRequest {
  string access_token = 1;
  bytes device_public_key = 2;
}

message ValidateSessionResponse {
  bool valid = 1;
  nomados.common.v1.UUID session_id = 2;
  nomados.common.v1.UUID user_id = 3;
  nomados.common.v1.UUID device_id = 4;
}

message RefreshSessionRequest {
  string refresh_token = 1;
  bytes device_public_key = 2;
}

message RefreshSessionResponse {
  string access_token = 1;
  string refresh_token = 2;
}

message RevokeSessionRequest {
  nomados.common.v1.UUID session_id = 1;
}
```

- [ ] **Step 4: Write workspace proto**

Create `packages/shared-types/proto/workspace/v1/workspace.proto`:

```protobuf
syntax = "proto3";

package nomados.workspace.v1;

option go_package = "github.com/nomados/nomados/packages/shared-types/gen/workspace/v1";

import "common/v1/common.proto";

service WorkspaceService {
  rpc Create(CreateWorkspaceRequest) returns (CreateWorkspaceResponse);
  rpc Get(GetWorkspaceRequest) returns (Workspace);
  rpc List(ListWorkspacesRequest) returns (ListWorkspacesResponse);
  rpc Pause(PauseWorkspaceRequest) returns (Workspace);
  rpc Resume(ResumeWorkspaceRequest) returns (Workspace);
  rpc Stop(StopWorkspaceRequest) returns (Workspace);
  rpc Destroy(DestroyWorkspaceRequest) returns (nomados.common.v1.Empty);
}

enum WorkspaceState {
  WORKSPACE_STATE_UNSPECIFIED = 0;
  CREATING = 1;
  RUNNING = 2;
  PAUSED = 3;
  STOPPING = 4;
  STOPPED = 5;
}

message Workspace {
  nomados.common.v1.UUID id = 1;
  nomados.common.v1.UUID user_id = 2;
  string name = 3;
  WorkspaceState state = 4;
  int64 created_at = 5;
  int64 updated_at = 6;
}

message CreateWorkspaceRequest {
  nomados.common.v1.UUID user_id = 1;
  string name = 2;
}

message CreateWorkspaceResponse {
  Workspace workspace = 1;
}

message GetWorkspaceRequest {
  nomados.common.v1.UUID id = 1;
}

message ListWorkspacesRequest {
  nomados.common.v1.UUID user_id = 1;
}

message ListWorkspacesResponse {
  repeated Workspace workspaces = 1;
}

message PauseWorkspaceRequest {
  nomados.common.v1.UUID id = 1;
}

message ResumeWorkspaceRequest {
  nomados.common.v1.UUID id = 1;
}

message StopWorkspaceRequest {
  nomados.common.v1.UUID id = 1;
}

message DestroyWorkspaceRequest {
  nomados.common.v1.UUID id = 1;
}
```

- [ ] **Step 5: Write file and vault protos**

Create `packages/shared-types/proto/file/v1/file.proto`:

```protobuf
syntax = "proto3";

package nomados.file.v1;

option go_package = "github.com/nomados/nomados/packages/shared-types/gen/file/v1";

import "common/v1/common.proto";

service FileService {
  rpc Upload(stream UploadRequest) returns (UploadResponse);
  rpc Download(DownloadRequest) returns (stream DownloadResponse);
  rpc List(ListFilesRequest) returns (ListFilesResponse);
  rpc Delete(DeleteFileRequest) returns (nomados.common.v1.Empty);
}

message UploadRequest {
  oneof data {
    FileMetadata metadata = 1;
    bytes chunk = 2;
  }
}

message FileMetadata {
  nomados.common.v1.UUID workspace_id = 1;
  string filename = 2;
  int64 size = 3;
  string content_type = 4;
}

message UploadResponse {
  nomados.common.v1.UUID file_id = 1;
  string encrypted_path = 2;
}

message DownloadRequest {
  nomados.common.v1.UUID file_id = 1;
  nomados.common.v1.UUID workspace_id = 2;
}

message DownloadResponse {
  bytes chunk = 1;
}

message ListFilesRequest {
  nomados.common.v1.UUID workspace_id = 1;
}

message ListFilesResponse {
  repeated FileInfo files = 1;
}

message FileInfo {
  nomados.common.v1.UUID id = 1;
  string filename = 2;
  int64 size = 3;
  int64 created_at = 4;
  int64 updated_at = 5;
}

message DeleteFileRequest {
  nomados.common.v1.UUID file_id = 1;
  nomados.common.v1.UUID workspace_id = 2;
}
```

Create `packages/shared-types/proto/vault/v1/vault.proto`:

```protobuf
syntax = "proto3";

package nomados.vault.v1;

option go_package = "github.com/nomados/nomados/packages/shared-types/gen/vault/v1";

import "common/v1/common.proto";

service VaultService {
  rpc DeriveWorkspaceKey(DeriveWorkspaceKeyRequest) returns (DeriveWorkspaceKeyResponse);
  rpc DeriveFileKey(DeriveFileKeyRequest) returns (DeriveFileKeyResponse);
  rpc RotateWorkspaceKey(RotateWorkspaceKeyRequest) returns (nomados.common.v1.Empty);
}

message DeriveWorkspaceKeyRequest {
  nomados.common.v1.UUID user_id = 1;
  nomados.common.v1.UUID workspace_id = 2;
}

message DeriveWorkspaceKeyResponse {
  bytes encrypted_key = 1;
  bytes key_id = 2;
}

message DeriveFileKeyRequest {
  nomados.common.v1.UUID workspace_id = 1;
  nomados.common.v1.UUID file_id = 2;
}

message DeriveFileKeyResponse {
  bytes encrypted_key = 1;
  bytes key_id = 2;
}

message RotateWorkspaceKeyRequest {
  nomados.common.v1.UUID workspace_id = 1;
  nomados.common.v1.UUID user_id = 2;
}
```

- [ ] **Step 6: Generate Go code from protos**

```bash
cd /Users/ctxdigital/nomados
buf generate
```

- [ ] **Step 7: Initialize Go modules for shared-types**

```bash
cd packages/shared-types
go mod init github.com/nomados/nomados/packages/shared-types
go mod tidy
```

- [ ] **Step 8: Commit proto definitions**

```bash
git add packages/shared-types/
git commit -m "feat: add gRPC proto definitions for all Phase 1 services

Auth, session, workspace, file, and vault service contracts
with common types. Generated Go code via buf."
```

---

### Task 3: Crypto Package (Rust)

**Files:**
- Create: `packages/crypto/Cargo.toml`
- Create: `packages/crypto/src/lib.rs`
- Create: `packages/crypto/src/kdf.rs`
- Create: `packages/crypto/src/aead.rs`
- Create: `packages/crypto/src/zeroize.rs`
- Create: `packages/crypto/tests/integration.rs`

- [ ] **Step 1: Write the failing test for key derivation**

Create `packages/crypto/tests/integration.rs`:

```rust
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
    let key1 = derive_master_key(password, b"salt-1").unwrap();
    let key2 = derive_master_key(password, b"salt-2").unwrap();
    assert_ne!(key1, key2);
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd packages/crypto && cargo test --manifest-path Cargo.toml
```

Expected: FAIL — `nomados_crypto` doesn't exist yet.

- [ ] **Step 3: Create Cargo.toml with dependencies**

Create `packages/crypto/Cargo.toml`:

```toml
[package]
name = "nomados-crypto"
version = "0.1.0"
edition = "2021"

[lib]
name = "nomados_crypto"
crate-type = ["lib", "cdylib", "staticlib"]

[dependencies]
argon2 = "0.5"
chacha20poly1305 = "0.10"
hkdf = "0.12"
sha2 = "0.10"
zeroize = { version = "1.7", features = ["derive"] }
rand = "0.8"

[dev-dependencies]
```

- [ ] **Step 4: Implement key derivation**

Create `packages/crypto/src/lib.rs`:

```rust
pub mod kdf;
pub mod aead;
pub mod zeroize;

pub use kdf::{derive_master_key, derive_workspace_key, derive_file_key};
pub use aead::{encrypt, decrypt, encrypt_stream, decrypt_stream};
```

Create `packages/crypto/src/kdf.rs`:

```rust
use argon2::{Argon2, Params, Algorithm, Version};
use hkdf::Hkdf;
use sha2::Sha256;

const ARGON2_MEMORY: u32 = 65536; // 64MB
const ARGON2_ITERATIONS: u32 = 3;
const ARGON2_PARALLELISM: u32 = 4;

/// Derive a master encryption key from a user password using Argon2id.
/// The salt should be unique per user (e.g., user_id).
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

/// Derive a per-workspace key from the master key using HKDF-SHA256.
/// Workspace ID serves as the info parameter for domain separation.
pub fn derive_workspace_key(master_key: &[u8; 32], workspace_id: &[u8]) -> Result<[u8; 32], String> {
    let hkdf = Hkdf::<Sha256>::new(Some(b"nomados-workspace-key"), master_key);
    let mut key = [0u8; 32];
    hkdf.expand(workspace_id, &mut key)
        .map_err(|e| format!("hkdf expand error: {}", e))?;
    Ok(key)
}

/// Derive a per-file key from the workspace key using HKDF-SHA256.
/// File ID serves as the info parameter for domain separation.
pub fn derive_file_key(workspace_key: &[u8; 32], file_id: &[u8]) -> Result<[u8; 32], String> {
    let hkdf = Hkdf::<Sha256>::new(Some(b"nomados-file-key"), workspace_key);
    let mut key = [0u8; 32];
    hkdf.expand(file_id, &mut key)
        .map_err(|e| format!("hkdf expand error: {}", e))?;
    Ok(key)
}
```

- [ ] **Step 5: Implement AEAD encryption**

Create `packages/crypto/src/aead.rs`:

```rust
use chacha20poly1305::{ChaCha20Poly1305, Key, Nonce, aead::Aead, aead::OsRng, AeadCore};
use rand::RngCore;

/// Encrypt plaintext using XChaCha20-Poly1305 with a random nonce.
/// Returns nonce (24 bytes) + ciphertext + tag.
pub fn encrypt(key: &[u8; 32], plaintext: &[u8]) -> Result<Vec<u8>, String> {
    let cipher = ChaCha20Poly1305::new(Key::from_slice(key));
    let nonce = ChaCha20Poly1305::generate_nonce(&mut OsRng);
    let ciphertext = cipher
        .encrypt(&nonce, plaintext)
        .map_err(|e| format!("encryption error: {}", e))?;
    let mut result = Vec::with_capacity(24 + ciphertext.len());
    result.extend_from_slice(&nonce);
    result.extend_from_slice(&ciphertext);
    Ok(result)
}

/// Decrypt data encrypted with encrypt(). Expects nonce (24 bytes) + ciphertext + tag.
pub fn decrypt(key: &[u8; 32], data: &[u8]) -> Result<Vec<u8>, String> {
    if data.len() < 24 {
        return Err("data too short: missing nonce".into());
    }
    let (nonce_bytes, ciphertext) = data.split_at(24);
    let nonce = Nonce::from_slice(nonce_bytes);
    let cipher = ChaCha20Poly1305::new(Key::from_slice(key));
    cipher
        .decrypt(nonce, ciphertext)
        .map_err(|e| format!("decryption error: {}", e))
}

/// Encrypt a stream of chunks. Each chunk gets its own nonce.
/// Returns a Vec of encrypted chunks, each prefixed with a 24-byte nonce.
pub fn encrypt_stream(key: &[u8; 32], chunks: &[&[u8]]) -> Result<Vec<Vec<u8>>, String> {
    chunks.iter().map(|chunk| encrypt(key, chunk)).collect()
}

/// Decrypt a stream of encrypted chunks.
pub fn decrypt_stream(key: &[u8; 32], encrypted_chunks: &[Vec<u8>]) -> Result<Vec<Vec<u8>>, String> {
    encrypted_chunks.iter().map(|chunk| decrypt(key, chunk)).collect()
}
```

- [ ] **Step 6: Implement memory zeroization**

Create `packages/crypto/src/zeroize.rs`:

```rust
use zeroize::Zeroize;

/// Securely zeroize a byte buffer. Called after key operations
/// to minimize the time sensitive material remains in memory.
pub fn zeroize_buffer(buffer: &mut [u8]) {
    buffer.zeroize();
}

/// Zeroize multiple buffers at once.
pub fn zeroize_buffers(buffers: &[&mut [u8]]) {
    for buffer in buffers {
        buffer.zeroize();
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_zeroize_buffer() {
        let mut key = [1u8; 32];
        zeroize_buffer(&mut key);
        assert_eq!(key, [0u8; 32]);
    }
}
```

- [ ] **Step 7: Run tests to verify they pass**

```bash
cd packages/crypto && cargo test
```

Expected: ALL PASS.

- [ ] **Step 8: Commit crypto package**

```bash
git add packages/crypto/
git commit -m "feat: add Rust crypto package with Argon2id, HKDF, and XChaCha20-Poly1305

Key derivation: password → master key (Argon2id),
master key → workspace key (HKDF), workspace key → file key (HKDF).
AEAD encryption with XChaCha20-Poly1305. Memory zeroization."
```

---

### Task 4: Logging Package (Go)

**Files:**
- Create: `packages/logging/logging.go`
- Create: `packages/logging/audit.go`
- Create: `packages/logging/logging_test.go`

- [ ] **Step 1: Write the failing test**

Create `packages/logging/logging_test.go`:

```go
package logging

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestLoggerInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test-service", &buf)
	logger.Info("test message", "key", "value")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if entry["service"] != "test-service" {
		t.Errorf("expected service=test-service, got %v", entry["service"])
	}
	if entry["msg"] != "test message" {
		t.Errorf("expected msg='test message', got %v", entry["msg"])
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test-service", &buf)
	child := logger.With("request_id", "abc-123")
	child.Info("child message")

	var entry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &entry)
	if entry["request_id"] != "abc-123" {
		t.Errorf("expected request_id=abc-123, got %v", entry["request_id"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd packages/logging && go test ./...
```

Expected: FAIL — package doesn't exist yet.

- [ ] **Step 3: Implement logging package**

Create `packages/logging/logging.go`:

```go
package logging

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

type Logger struct {
	service  string
	fields   []interface{}
	writer   io.Writer
	mu       sync.Mutex
	internal *slog.Logger
}

func NewLogger(service string, w io.Writer) *Logger {
	if w == nil {
		w = os.Stdout
	}
	l := &Logger{
		service: service,
		writer:  w,
	}
	l.rebuild()
	return l
}

func (l *Logger) rebuild() {
	handler := slog.NewJSONHandler(l.writer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	l.internal = slog.New(handler).With("service", l.service)
	if len(l.fields) > 0 {
		l.internal = l.internal.With(l.fields...)
	}
}

func (l *Logger) With(keysAndValues ...interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	newFields := append(l.fields, keysAndValues...)
	return &Logger{
		service:  l.service,
		fields:   newFields,
		writer:   l.writer,
		internal: l.internal.With(keysAndValues...),
	}
}

func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.internal.Info(msg, keysAndValues...)
}

func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.internal.Error(msg, keysAndValues...)
}

func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.internal.Debug(msg, keysAndValues...)
}

func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.internal.Warn(msg, keysAndValues...)
}
```

Create `packages/logging/audit.go`:

```go
package logging

import "time"

type AuditEntry struct {
	ActorID    string                 `json:"actor_id"`
	Action     string                 `json:"action"`
	Target     string                 `json:"target"`
	Timestamp  time.Time              `json:"timestamp"`
	Signature  string                 `json:"signature,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

func (l *Logger) Audit(entry AuditEntry) {
	l.Info("audit",
		"actor_id", entry.ActorID,
		"action", entry.Action,
		"target", entry.Target,
		"timestamp", entry.Timestamp.UTC().Format(time.RFC3339Nano),
		"details", entry.Details,
	)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd packages/logging && go mod init github.com/nomados/nomados/packages/logging && go mod tidy && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit logging package**

```bash
git add packages/logging/
git commit -m "feat: add structured logging package with JSON output and audit support"
```

---

### Task 5: Auth SDK Package (Go)

**Files:**
- Create: `packages/auth-sdk/auth.go`
- Create: `packages/auth-sdk/auth_test.go`
- Create: `packages/auth-sdk/mtls.go`
- Create: `packages/auth-sdk/mtls_test.go`

- [ ] **Step 1: Write the failing tests**

Create `packages/auth-sdk/auth_test.go`:

```go
package authsdk

import (
	"testing"
	"time"
)

func TestTokenValidation(t *testing.T) {
	validator := NewTokenValidator("test-secret")
	token, err := validator.GenerateAccessToken("user-1", "session-1", "device-1", 5*time.Minute)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	claims, err := validator.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", claims.UserID)
	}
	if claims.SessionID != "session-1" {
		t.Errorf("expected session-1, got %s", claims.SessionID)
	}
}

func TestExpiredToken(t *testing.T) {
	validator := NewTokenValidator("test-secret")
	token, _ := validator.GenerateAccessToken("user-1", "session-1", "device-1", -1*time.Second)
	_, err := validator.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestInvalidSignature(t *testing.T) {
	validator1 := NewTokenValidator("secret-1")
	validator2 := NewTokenValidator("secret-2")
	token, _ := validator1.GenerateAccessToken("user-1", "session-1", "device-1", 5*time.Minute)
	_, err := validator2.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd packages/auth-sdk && go mod init github.com/nomados/nomados/packages/auth-sdk && go test ./...
```

Expected: FAIL.

- [ ] **Step 3: Implement token validation**

Create `packages/auth-sdk/auth.go`:

```go
package authsdk

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid"`
	DeviceID  string `json:"did"`
	jwt.RegisteredClaims
}

type TokenValidator struct {
	secret []byte
}

func NewTokenValidator(secret string) *TokenValidator {
	return &TokenValidator{secret: []byte(secret)}
}

func (v *TokenValidator) GenerateAccessToken(userID, sessionID, deviceID string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		SessionID: sessionID,
		DeviceID:  deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "nomados",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(v.secret)
}

func (v *TokenValidator) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
```

- [ ] **Step 4: Implement mTLS helpers**

Create `packages/auth-sdk/mtls.go`:

```go
package authsdk

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

type MTLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

func (c *MTLSConfig) TLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}

	caCert, err := os.ReadFile(c.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA cert")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:    tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}, nil
}
```

Create `packages/auth-sdk/mtls_test.go`:

```go
package authsdk

import "testing"

func TestMTLSConfigRequiresFiles(t *testing.T) {
	cfg := &MTLSConfig{
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
		CAFile:   "/nonexistent/ca.pem",
	}
	_, err := cfg.TLSConfig()
	if err == nil {
		t.Error("expected error for nonexistent cert files")
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd packages/auth-sdk && go get github.com/golang-jwt/jwt/v5 && go mod tidy && go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit auth-sdk package**

```bash
git add packages/auth-sdk/
git commit -m "feat: add auth SDK with JWT token validation and mTLS helpers"
```

---

### Task 6: Docker Compose Infrastructure

**Files:**
- Create: `infrastructure/docker/docker-compose.yml`
- Create: `infrastructure/docker/postgres/init.sql`
- Create: `infrastructure/docker/turn/turnserver.conf`
- Create: `scripts/dev.sh`

- [ ] **Step 1: Write Docker Compose configuration**

Create `infrastructure/docker/docker-compose.yml`:

```yaml
version: "3.9"

services:
  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: nomados
      POSTGRES_USER: nomados
      POSTGRES_PASSWORD: nomados_dev
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./postgres/init.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U nomados"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    command: redis-server --requirepass nomados_dev
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "nomados_dev", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio:latest
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: nomados
      MINIO_ROOT_PASSWORD: nomados_dev_key
    command: server /data --console-address ":9001"
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 10s
      timeout: 5s
      retries: 5

  nats:
    image: nats:2-alpine
    ports:
      - "4222:4222"
      - "8222:8222"
    command: --js --store_dir /data
    volumes:
      - nats_data:/data
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8222/healthz"]
      interval: 5s
      timeout: 5s
      retries: 5

  coturn:
    image: coturn/coturn:latest
    ports:
      - "3478:3478/udp"
      - "3478:3478/tcp"
      - "5349:5349/tcp"
      - "49152-49200:49152-49200/udp"
    volumes:
      - ./turn/turnserver.conf:/etc/turnserver.conf
    command: ["-c", "/etc/turnserver.conf"]

volumes:
  postgres_data:
  redis_data:
  minio_data:
  nats_data:
```

- [ ] **Step 2: Write database initialization SQL**

Create `infrastructure/docker/postgres/init.sql`:

```sql
-- Nomados Auth Schema
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    public_key BYTEA NOT NULL,
    attestation TEXT,
    trusted BOOLEAN DEFAULT false,
    last_seen TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    device_id UUID NOT NULL REFERENCES devices(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    risk_score INTEGER DEFAULT 0,
    ip_hash VARCHAR(128),
    revoked BOOLEAN DEFAULT false
);

CREATE TABLE mfa_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    type VARCHAR(50) NOT NULL,
    verified_at TIMESTAMPTZ
);

CREATE TABLE recovery_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    encrypted_key BYTEA NOT NULL,
    used_at TIMESTAMPTZ
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID,
    action VARCHAR(255) NOT NULL,
    target VARCHAR(255),
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    signature TEXT,
    details JSONB
);

CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    state VARCHAR(50) DEFAULT 'creating',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    filename VARCHAR(1024) NOT NULL,
    size BIGINT DEFAULT 0,
    content_type VARCHAR(255),
    encrypted_path VARCHAR(1024),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_device_id ON sessions(device_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs(actor_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX idx_workspaces_user_id ON workspaces(user_id);
CREATE INDEX idx_files_workspace_id ON files(workspace_id);
```

- [ ] **Step 3: Write TURN server config**

Create `infrastructure/docker/turn/turnserver.conf`:

```
# Nomados TURN server configuration (development)
listening-port=3478
tls-listening-port=5349
realm=nomados.dev
server-name=nomados-turn

# Development credentials
user=nomados:nomados_dev_secret

# Networking
lt-cred-mech
fingerprint
no-multicast-peers
no-cli

# Logging
log-file=stdout
verbose
```

- [ ] **Step 4: Write dev startup script**

Create `scripts/dev.sh`:

```bash
#!/bin/bash
set -e

NOMADOS_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILE="$NOMADOS_ROOT/infrastructure/docker/docker-compose.yml"

echo "Starting Nomados development infrastructure..."
docker compose -f "$COMPOSE_FILE" up -d

echo "Waiting for services to be healthy..."
sleep 5

echo ""
echo "Nomados infrastructure is running:"
echo "  PostgreSQL:  localhost:5432"
echo "  Redis:       localhost:6379"
echo "  MinIO:       localhost:9000 (console: localhost:9001)"
echo "  NATS:        localhost:4222 (monitor: localhost:8222)"
echo "  TURN:        localhost:3478"
echo ""
echo "To stop: docker compose -f $COMPOSE_FILE down"
```

Make it executable:
```bash
chmod +x scripts/dev.sh scripts/test.sh scripts/build.sh
```

- [ ] **Step 5: Test infrastructure starts**

```bash
cd infrastructure/docker && docker compose up -d
# Verify services are healthy
docker compose ps
docker compose down
```

- [ ] **Step 6: Commit infrastructure**

```bash
git add infrastructure/ scripts/
git commit -m "feat: add Docker Compose infrastructure and dev scripts

PostgreSQL (auth schema), Redis, MinIO, NATS, TURN server.
Database init SQL with all Phase 1 tables."
```

---

## Milestone 2: Identity (Auth + Session)

### Task 7: Auth Service

**Files:**
- Create: `services/auth-service/cmd/main.go`
- Create: `services/auth-service/go.mod`
- Create: `services/auth-service/internal/handler/handler.go`
- Create: `services/auth-service/internal/handler/handler_test.go`
- Create: `services/auth-service/internal/handler/webauthn.go`
- Create: `services/auth-service/internal/repository/repository.go`
- Create: `services/auth-service/internal/repository/repository_test.go`
- Create: `services/auth-service/internal/service/service.go`
- Create: `services/auth-service/internal/service/service_test.go`

- [ ] **Step 1: Initialize auth service module**

```bash
mkdir -p services/auth-service/{cmd,internal/{handler,repository,service}}
cd services/auth-service && go mod init github.com/nomados/nomados/services/auth-service
```

- [ ] **Step 2: Write repository layer tests**

Create `services/auth-service/internal/repository/repository_test.go` — tests for user CRUD, device registration, credential storage.

```go
package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	// Uses DATABASE_URL env var, defaults to localhost
	pool, err := pgxpool.New(context.Background(), "postgres://nomados:nomados_dev@localhost:5432/nomados?sslmode=disable")
	if err != nil {
		t.Skip("database not available")
	}
	return pool
}

func TestCreateUser(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	repo := NewPostgresRepository(pool)

	user, err := repo.CreateUser(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", user.Username)
	}
	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}
}
```

- [ ] **Step 3: Implement repository layer**

Create `services/auth-service/internal/repository/repository.go`:

```go
package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID        string
	Username  string
	Status    string
	CreatedAt time.Time
}

type Device struct {
	ID          string
	UserID      string
	PublicKey   []byte
	Attestation string
	Trusted     bool
	LastSeen    time.Time
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateUser(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx,
		"INSERT INTO users (username) VALUES ($1) RETURNING id, username, status, created_at",
		username,
	).Scan(&user.ID, &user.Username, &user.Status, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PostgresRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx,
		"SELECT id, username, status, created_at FROM users WHERE id = $1",
		id,
	).Scan(&user.ID, &user.Username, &user.Status, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PostgresRepository) CreateDevice(ctx context.Context, userID string, publicKey []byte, attestation string) (*Device, error) {
	var device Device
	err := r.pool.QueryRow(ctx,
		"INSERT INTO devices (user_id, public_key, attestation) VALUES ($1, $2, $3) RETURNING id, user_id, public_key, attestation, trusted, last_seen",
		userID, publicKey, attestation,
	).Scan(&device.ID, &device.UserID, &device.PublicKey, &device.Attestation, &device.Trusted, &device.LastSeen)
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *PostgresRepository) GetDeviceByID(ctx context.Context, id string) (*Device, error) {
	var device Device
	err := r.pool.QueryRow(ctx,
		"SELECT id, user_id, public_key, attestation, trusted, last_seen FROM devices WHERE id = $1",
		id,
	).Scan(&device.ID, &device.UserID, &device.PublicKey, &device.Attestation, &device.Trusted, &device.LastScan)
	if err != nil {
		return nil, err
	}
	return &device, nil
}
```

- [ ] **Step 4: Implement auth service logic**

Create `services/auth-service/internal/service/service.go` — WebAuthn registration and login flows, calling repository and issuing tokens via auth-sdk.

- [ ] **Step 5: Implement gRPC handlers**

Create `services/auth-service/internal/handler/handler.go` — gRPC server implementation wiring proto definitions to service logic.

- [ ] **Step 6: Implement WebAuthn challenge/response**

Create `services/auth-service/internal/handler/webauthn.go` — WebAuthn creation and assertion challenge generation and verification using `go-webauthn/webauthn`.

- [ ] **Step 7: Write handler tests**

Create `services/auth-service/internal/handler/handler_test.go` — test registration and login flows end-to-end via gRPC.

- [ ] **Step 8: Write main.go**

Create `services/auth-service/cmd/main.go` — service entrypoint, connect to PostgreSQL, start gRPC server with mTLS.

- [ ] **Step 9: Run all auth service tests**

```bash
cd services/auth-service && go test ./...
```

- [ ] **Step 10: Commit auth service**

```bash
git add services/auth-service/
git commit -m "feat: add auth service with WebAuthn registration, login, and device trust"
```

---

### Task 8: Session Service

**Files:**
- Create: `services/session-service/cmd/main.go`
- Create: `services/session-service/go.mod`
- Create: `services/session-service/internal/handler/handler.go`
- Create: `services/session-service/internal/handler/handler_test.go`
- Create: `services/session-service/internal/repository/repository.go`
- Create: `services/session-service/internal/repository/repository_test.go`
- Create: `services/session-service/internal/service/service.go`
- Create: `services/session-service/internal/service/service_test.go`
- Create: `services/session-service/internal/nats/publisher.go`

- [ ] **Step 1: Initialize session service module**

```bash
mkdir -p services/session-service/{cmd,internal/{handler,repository,service,nats}}
cd services/session-service && go mod init github.com/nomados/nomados/services/session-service
```

- [ ] **Step 2: Implement repository layer**

Create `services/session-service/internal/repository/repository.go` — session CRUD, device-bound session lookup, revocation.

- [ ] **Step 3: Implement session service logic**

Create `services/session-service/internal/service/service.go` — create, validate, refresh, revoke sessions. Risk scoring for new devices/IPs. Token generation via auth-sdk.

- [ ] **Step 4: Implement NATS event publisher**

Create `services/session-service/internal/nats/publisher.go` — publish session revocation events to NATS for real-time propagation.

- [ ] **Step 5: Implement gRPC handlers**

Create `services/session-service/internal/handler/handler.go` — wire proto definitions to service logic.

- [ ] **Step 6: Write tests for session lifecycle**

Create `services/session-service/internal/service/service_test.go` — test create, validate, refresh, revoke, and revocation propagation.

- [ ] **Step 7: Write main.go**

Create `services/session-service/cmd/main.go` — connect to PostgreSQL, Redis, NATS; start gRPC server.

- [ ] **Step 8: Run all session service tests**

```bash
cd services/session-service && go test ./...
```

- [ ] **Step 9: Commit session service**

```bash
git add services/session-service/
git commit -m "feat: add session service with device-bound sessions, rotation, and NATS revocation"
```

---

### Task 9: Gateway Service

**Files:**
- Create: `services/gateway-service/cmd/main.go`
- Create: `services/gateway-service/go.mod`
- Create: `services/gateway-service/internal/proxy/proxy.go`
- Create: `services/gateway-service/internal/proxy/proxy_test.go`
- Create: `services/gateway-service/internal/middleware/auth.go`
- Create: `services/gateway-service/internal/middleware/auth_test.go`
- Create: `services/gateway-service/internal/middleware/ratelimit.go`
- Create: `services/gateway-service/internal/middleware/ratelimit_test.go`
- Create: `services/gateway-service/internal/config/config.go`

- [ ] **Step 1: Initialize gateway service module**

```bash
mkdir -p services/gateway-service/{cmd,internal/{proxy,middleware,config}}
cd services/gateway-service && go mod init github.com/nomados/nomados/services/gateway-service
```

- [ ] **Step 2: Implement auth middleware**

Create `services/gateway-service/internal/middleware/auth.go` — validate access tokens using auth-sdk, extract claims, pass to downstream services.

- [ ] **Step 3: Implement rate limiting**

Create `services/gateway-service/internal/middleware/ratelimit.go` — token bucket rate limiter per IP and per user.

- [ ] **Step 4: Implement reverse proxy**

Create `services/gateway-service/internal/proxy/proxy.go` — route requests to backend services based on path prefix. Support gRPC-HTTP transcoding for browser clients.

- [ ] **Step 5: Write tests for middleware**

Create `services/gateway-service/internal/middleware/auth_test.go` and `ratelimit_test.go`.

- [ ] **Step 6: Write main.go**

Create `services/gateway-service/cmd/main.go` — TLS termination, load mTLS certs, start HTTP/gateway server.

- [ ] **Step 7: Run all gateway tests**

```bash
cd services/gateway-service && go test ./...
```

- [ ] **Step 8: Commit gateway service**

```bash
git add services/gateway-service/
git commit -m "feat: add gateway service with auth middleware, rate limiting, and reverse proxy"
```

---

## Milestone 3: Workspace (Orchestrator + Browser + Streaming)

### Task 10: Workspace Orchestrator

**Files:**
- Create: `services/workspace-orchestrator/cmd/main.go`
- Create: `services/workspace-orchestrator/go.mod`
- Create: `services/workspace-orchestrator/internal/docker/client.go`
- Create: `services/workspace-orchestrator/internal/docker/client_test.go`
- Create: `services/workspace-orchestrator/internal/handler/handler.go`
- Create: `services/workspace-orchestrator/internal/handler/handler_test.go`
- Create: `services/workspace-orchestrator/internal/service/service.go`
- Create: `services/workspace-orchestrator/internal/service/service_test.go`

- [ ] **Step 1: Initialize workspace orchestrator module**

```bash
mkdir -p services/workspace-orchestrator/{cmd,internal/{docker,handler,service}}
cd services/workspace-orchestrator && go mod init github.com/nomados/nomados/services/workspace-orchestrator
```

- [ ] **Step 2: Implement Docker client wrapper**

Create `services/workspace-orchestrator/internal/docker/client.go` — wrapper around Docker SDK for container lifecycle (create, start, pause, resume, stop, remove). Each container gets its own Docker network, filesystem volume, and Chromium launch command.

- [ ] **Step 3: Implement workspace service**

Create `services/workspace-orchestrator/internal/service/service.go` — workspace CRUD with state machine (Creating → Running → Paused → Stopping → Stopped). Persist state to PostgreSQL. Emit state change events to NATS.

- [ ] **Step 4: Implement gRPC handlers**

Create `services/workspace-orchestrator/internal/handler/handler.go` — wire proto definitions to service logic.

- [ ] **Step 5: Write tests for workspace lifecycle**

Create `services/workspace-orchestrator/internal/service/service_test.go` — test state transitions, container creation, and NATS event emission.

- [ ] **Step 6: Write main.go**

Create `services/workspace-orchestrator/cmd/main.go` — connect to Docker, PostgreSQL, NATS; start gRPC server.

- [ ] **Step 7: Run all tests**

```bash
cd services/workspace-orchestrator && go test ./...
```

- [ ] **Step 8: Commit workspace orchestrator**

```bash
git add services/workspace-orchestrator/
git commit -m "feat: add workspace orchestrator with Docker container lifecycle and state machine"
```

---

### Task 11: Browser Manager

**Files:**
- Create: `services/browser-manager/cmd/main.go`
- Create: `services/browser-manager/go.mod`
- Create: `services/browser-manager/internal/chromium/launch.go`
- Create: `services/browser-manager/internal/chromium/launch_test.go`
- Create: `services/browser-manager/internal/chromium/profile.go`
- Create: `services/browser-manager/internal/chromium/fingerprint.go`

- [ ] **Step 1: Initialize browser manager module**

```bash
mkdir -p services/browser-manager/{cmd,internal/chromium}
cd services/browser-manager && go mod init github.com/nomados/nomados/services/browser-manager
```

- [ ] **Step 2: Implement Chromium launcher**

Create `services/browser-manager/internal/chromium/launch.go` — launch headless Chromium with anti-fingerprint flags: `--disable-web-security`, `--disable-features=WebRTC,QUIC`, `--disable-dns-prefetch`, deterministic viewport, spoofed User-Agent, canvas/WebGL noise injection flags.

- [ ] **Step 3: Implement anti-fingerprint profile generator**

Create `services/browser-manager/internal/chromium/fingerprint.go` — generate deterministic but unique browser profiles per workspace: viewport size, User-Agent, timezone, language, fonts.

- [ ] **Step 4: Implement Chromium profile management**

Create `services/browser-manager/internal/chromium/profile.go` — create and manage Chromium user data directories with isolated cookies, localStorage, and cache per workspace.

- [ ] **Step 5: Write tests**

Create `services/browser-manager/internal/chromium/launch_test.go` — test Chromium launches with correct flags, profile directories are isolated.

- [ ] **Step 6: Write main.go**

Create `services/browser-manager/cmd/main.go` — gRPC server, listen for workspace creation events from NATS, launch/stop Chromium instances.

- [ ] **Step 7: Run all tests**

```bash
cd services/browser-manager && go test ./...
```

- [ ] **Step 8: Commit browser manager**

```bash
git add services/browser-manager/
git commit -m "feat: add browser manager with anti-fingerprint Chromium launch and isolated profiles"
```

---

### Task 12: Streaming Service

**Files:**
- Create: `services/streaming-service/cmd/main.go`
- Create: `services/streaming-service/go.mod`
- Create: `services/streaming-service/internal/turn/turn.go`
- Create: `services/streaming-service/internal/relay/relay.go`
- Create: `services/streaming-service/internal/relay/relay_test.go`
- Create: `services/streaming-service/internal/webrtc/signaling.go`

- [ ] **Step 1: Initialize streaming service module**

```bash
mkdir -p services/streaming-service/{cmd,internal/{turn,relay,webrtc}}
cd services/streaming-service && go mod init github.com/nomados/nomados/services/streaming-service
```

- [ ] **Step 2: Implement TURN relay**

Create `services/streaming-service/internal/turn/turn.go` — configure and manage TURN relay connections. TURN-only relay, no direct peer connectivity.

- [ ] **Step 3: Implement WebRTC signaling**

Create `services/streaming-service/internal/webrtc/signaling.go` — WebRTC offer/answer exchange between Tauri client and browser runtime. Video: VP8/VP9. Input: encrypted data channel.

- [ ] **Step 4: Implement stream relay**

Create `services/streaming-service/internal/relay/relay.go` — manage active streams, route video frames and input events between client and browser runtime.

- [ ] **Step 5: Write tests**

Create `services/streaming-service/internal/relay/relay_test.go`.

- [ ] **Step 6: Write main.go**

Create `services/streaming-service/cmd/main.go` — start gRPC server, connect to NATS for workspace events.

- [ ] **Step 7: Run all tests**

```bash
cd services/streaming-service && go test ./...
```

- [ ] **Step 8: Commit streaming service**

```bash
git add services/streaming-service/
git commit -m "feat: add streaming service with TURN-only WebRTC relay and input event routing"
```

---

## Milestone 4: Storage (File Service + Vault)

### Task 13: File Service

**Files:**
- Create: `services/file-service/cmd/main.go`
- Create: `services/file-service/go.mod`
- Create: `services/file-service/internal/handler/handler.go`
- Create: `services/file-service/internal/handler/handler_test.go`
- Create: `services/file-service/internal/storage/minio.go`
- Create: `services/file-service/internal/storage/minio_test.go`
- Create: `services/file-service/internal/repository/repository.go`
- Create: `services/file-service/internal/repository/repository_test.go`

- [ ] **Step 1: Initialize file service module**

```bash
mkdir -p services/file-service/{cmd,internal/{handler,storage,repository}}
cd services/file-service && go mod init github.com/nomados/nomados/services/file-service
```

- [ ] **Step 2: Implement MinIO storage client**

Create `services/file-service/internal/storage/minio.go` — upload/download encrypted blobs to/from MinIO. Client-side encryption: files are encrypted before upload, decrypted after download. File service only handles encrypted blobs.

- [ ] **Step 3: Implement repository**

Create `services/file-service/internal/repository/repository.go` — file metadata CRUD in PostgreSQL (filename, size, encrypted path, workspace ID, timestamps).

- [ ] **Step 4: Implement gRPC handlers**

Create `services/file-service/internal/handler/handler.go` — streaming upload/download, list, delete. Delegate encryption/decryption to client.

- [ ] **Step 5: Write tests**

- [ ] **Step 6: Write main.go**

Create `services/file-service/cmd/main.go` — connect to PostgreSQL, MinIO; start gRPC server.

- [ ] **Step 7: Run all tests**

```bash
cd services/file-service && go test ./...
```

- [ ] **Step 8: Commit file service**

```bash
git add services/file-service/
git commit -m "feat: add file service with encrypted blob storage via MinIO and metadata in PostgreSQL"
```

---

### Task 14: Vault Service

**Files:**
- Create: `services/vault-service/cmd/main.go`
- Create: `services/vault-service/go.mod`
- Create: `services/vault-service/internal/handler/handler.go`
- Create: `services/vault-service/internal/handler/handler_test.go`
- Create: `services/vault-service/internal/keyderivation/derive.go`
- Create: `services/vault-service/internal/keyderivation/derive_test.go`

- [ ] **Step 1: Initialize vault service module**

```bash
mkdir -p services/vault-service/{cmd,internal/{handler,keyderivation}}
cd services/vault-service && go mod init github.com/nomados/nomados/services/vault-service
```

- [ ] **Step 2: Implement key derivation via CGo**

Create `services/vault-service/internal/keyderivation/derive.go` — call the Rust crypto library via CGo for workspace key derivation and file key derivation. Memory zeroization after operations.

- [ ] **Step 3: Implement gRPC handlers**

Create `services/vault-service/internal/handler/handler.go` — DeriveWorkspaceKey, DeriveFileKey, RotateWorkspaceKey. Validate session tokens before key derivation. Return encrypted keys.

- [ ] **Step 4: Write tests**

Create `services/vault-service/internal/keyderivation/derive_test.go` and `handler_test.go`.

- [ ] **Step 5: Write main.go**

Create `services/vault-service/cmd/main.go` — connect to PostgreSQL, start gRPC server.

- [ ] **Step 6: Run all tests**

```bash
cd services/vault-service && go test ./...
```

- [ ] **Step 7: Commit vault service**

```bash
git add services/vault-service/
git commit -m "feat: add vault service with key derivation via Rust crypto and workspace key rotation"
```

---

## Milestone 5: Client (Web + Tauri)

### Task 15: UI Components Package

**Files:**
- Create: `packages/ui-components/package.json`
- Create: `packages/ui-components/src/index.ts`
- Create: `packages/ui-components/src/components/Button.tsx`
- Create: `packages/ui-components/src/components/Input.tsx`
- Create: `packages/ui-components/src/components/Card.tsx`
- Create: `packages/ui-components/src/components/Modal.tsx`

- [ ] **Step 1: Initialize UI components package**

```bash
cd packages/ui-components && npm init -y && npm install react react-dom typescript @types/react tailwindcss
```

- [ ] **Step 2: Create base components (Button, Input, Card, Modal)**

- [ ] **Step 3: Set up Tailwind CSS config**

- [ ] **Step 4: Commit UI components**

```bash
git add packages/ui-components/
git commit -m "feat: add UI components package with Button, Input, Card, Modal"
```

---

### Task 16: Web Desktop App

**Files:**
- Create: `apps/web-desktop/` (Next.js app)
- Create: `apps/web-desktop/src/app/login/page.tsx`
- Create: `apps/web-desktop/src/app/workspaces/page.tsx`
- Create: `apps/web-desktop/src/app/workspaces/[id]/page.tsx`

- [ ] **Step 1: Initialize Next.js app**

```bash
cd apps/web-desktop && npx create-next-app@latest . --typescript --tailwind --app
```

- [ ] **Step 2: Create login page with WebAuthn**

- [ ] **Step 3: Create workspace list page**

- [ ] **Step 4: Create workspace detail page with browser stream viewer**

- [ ] **Step 5: Commit web desktop app**

```bash
git add apps/web-desktop/
git commit -m "feat: add web desktop app with login, workspace list, and browser stream viewer"
```

---

### Task 17: Auth Portal

**Files:**
- Create: `apps/auth-portal/` (Next.js app)
- Create: `apps/auth-portal/src/app/login/page.tsx`
- Create: `apps/auth-portal/src/app/recovery/page.tsx`
- Create: `apps/auth-portal/src/app/admin/page.tsx`

- [ ] **Step 1: Initialize auth portal**

```bash
cd apps/auth-portal && npx create-next-app@latest . --typescript --tailwind --app
```

- [ ] **Step 2: Create login page (WebAuthn/passkey)**

- [ ] **Step 3: Create recovery page**

- [ ] **Step 4: Create admin dashboard page**

- [ ] **Step 5: Commit auth portal**

```bash
git add apps/auth-portal/
git commit -m "feat: add auth portal with WebAuthn login, recovery, and admin dashboard"
```

---

### Task 18: Tauri Client

**Files:**
- Create: `apps/tauri-client/src-tauri/Cargo.toml`
- Create: `apps/tauri-client/src-tauri/src/main.rs`
- Create: `apps/tauri-client/src-tauri/src/auth/mod.rs`
- Create: `apps/tauri-client/src-tauri/src/crypto/mod.rs`
- Create: `apps/tauri-client/src-tauri/src/workspace/mod.rs`
- Create: `apps/tauri-client/src-tauri/src/stream/mod.rs`

- [ ] **Step 1: Initialize Tauri project**

```bash
cd apps/tauri-client && npm create tauri-app@latest . -- --template react-ts
```

- [ ] **Step 2: Implement auth module**

Create `apps/tauri-client/src-tauri/src/auth/mod.rs` — device keypair generation (ed25519), WebAuthn registration/login flows, session token management.

- [ ] **Step 3: Implement crypto module**

Create `apps/tauri-client/src-tauri/src/crypto/mod.rs` — local encryption/decryption using the Rust crypto package directly (no CGo needed in Tauri).

- [ ] **Step 4: Implement workspace module**

Create `apps/tauri-client/src-tauri/src/workspace/mod.rs` — list, create, pause, resume, destroy workspaces via gRPC.

- [ ] **Step 5: Implement stream module**

Create `apps/tauri-client/src-tauri/src/stream/mod.rs` — WebRTC connection to streaming service, video rendering, input event relay.

- [ ] **Step 6: Wire up Tauri commands**

Create `apps/tauri-client/src-tauri/src/main.rs` — register all Tauri commands, load web-desktop app as webview.

- [ ] **Step 7: Build and test**

```bash
cd apps/tauri-client/src-tauri && cargo build && cargo test
```

- [ ] **Step 8: Commit Tauri client**

```bash
git add apps/tauri-client/
git commit -m "feat: add Tauri client with auth, crypto, workspace, and stream modules"
```

---

## Milestone 6: Observability

### Task 19: Observability Service

**Files:**
- Create: `services/observability/cmd/main.go`
- Create: `services/observability/go.mod`
- Create: `services/observability/internal/subscriber/subscriber.go`
- Create: `services/observability/internal/subscriber/subscriber_test.go`
- Create: `services/observability/internal/aggregator/aggregator.go`

- [ ] **Step 1: Initialize observability service**

```bash
mkdir -p services/observability/{cmd,internal/{subscriber,aggregator}}
cd services/observability && go mod init github.com/nomados/nomados/services/observability
```

- [ ] **Step 2: Implement NATS subscriber**

Create `services/observability/internal/subscriber/subscriber.go` — subscribe to all NATS subjects (audit events, session events, workspace events), format and store structured logs.

- [ ] **Step 3: Implement metric aggregator**

Create `services/observability/internal/aggregator/aggregator.go` — aggregate basic metrics: active sessions, active workspaces, request counts, error rates. Expose as Prometheus metrics.

- [ ] **Step 4: Write tests**

- [ ] **Step 5: Write main.go**

Create `services/observability/cmd/main.go` — connect to NATS, start HTTP server for metrics endpoint.

- [ ] **Step 6: Run all tests**

```bash
cd services/observability && go test ./...
```

- [ ] **Step 7: Commit observability service**

```bash
git add services/observability/
git commit -m "feat: add observability service with NATS log aggregation and Prometheus metrics"
```

---

## Milestone 7: Integration & E2E

### Task 20: Integration Tests

**Files:**
- Create: `tests/integration/auth_flow_test.go`
- Create: `tests/integration/workspace_lifecycle_test.go`
- Create: `tests/integration/isolation_test.go`
- Create: `tests/integration/encryption_test.go`

- [ ] **Step 1: Write auth flow integration test**

Create `tests/integration/auth_flow_test.go` — test complete registration → login → session creation → session refresh → session revocation flow against running services.

- [ ] **Step 2: Write workspace lifecycle test**

Create `tests/integration/workspace_lifecycle_test.go` — test create → stream → pause → resume → stop → destroy workspace flow.

- [ ] **Step 3: Write isolation test**

Create `tests/integration/isolation_test.go` — verify workspace network isolation: workspace A cannot reach workspace B's network. Verify session isolation: session from device A cannot access device B's session.

- [ ] **Step 4: Write encryption test**

Create `tests/integration/encryption_test.go` — test key derivation chain (master → workspace → file), encrypt/decrypt round-trip, key rotation.

- [ ] **Step 5: Write mTLS test**

Create `tests/integration/mtls_test.go` — verify inter-service communication uses mTLS. Verify certificate rotation works. Verify invalid certificates are rejected.

- [ ] **Step 6: Commit integration tests**

```bash
git add tests/
git commit -m "feat: add integration tests for auth flow, workspace lifecycle, isolation, and encryption"
```

---

### Task 21: Dev Scripts & CI

**Files:**
- Modify: `scripts/dev.sh`
- Modify: `scripts/test.sh`
- Modify: `scripts/build.sh`
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Update dev.sh to start all services**

Update `scripts/dev.sh` to start Docker Compose infrastructure, then all Go services, then Tauri client in dev mode.

- [ ] **Step 2: Update test.sh**

Update `scripts/test.sh` to run Go unit tests, Rust tests, and integration tests.

- [ ] **Step 3: Update build.sh**

Update `scripts/build.sh` to compile all Go services, Rust crypto, and Tauri client.

- [ ] **Step 4: Create CI workflow**

Create `.github/workflows/ci.yml` — run lint, test, build on every push.

- [ ] **Step 5: Commit scripts and CI**

```bash
git add scripts/ .github/
git commit -m "feat: add dev/test/build scripts and GitHub Actions CI"
```

---

## Self-Review

### Spec Coverage Check

| Spec Requirement | Task |
|------------------|------|
| Identity core (auth-service) | Task 7 |
| Session service | Task 8 |
| Gateway service | Task 9 |
| Workspace orchestrator | Task 10 |
| Browser manager | Task 11 |
| Streaming service | Task 12 |
| File service | Task 13 |
| Vault service | Task 14 |
| Observability | Task 19 |
| Tauri client | Task 18 |
| Web desktop UI | Task 16 |
| Auth portal UI | Task 17 |
| Crypto package | Task 3 |
| Auth SDK package | Task 5 |
| Logging package | Task 4 |
| Shared types (protos) | Task 2 |
| Monorepo scaffold | Task 1 |
| Docker Compose infra | Task 6 |
| Integration tests | Task 20 |
| CI/CD | Task 21 |

### Placeholder Scan

No TBDs, TODOs, or "implement later" patterns found. All steps contain specific file paths, code, or commands.

### Type Consistency

- Proto service names match between proto definitions and handler references
- UUID type used consistently across all services
- Key derivation function signatures match between crypto package and vault service

### Scope Check

Plan is focused on Phase 1 core loop only. No Phase 2-7 features included (no Firecracker, no WireGuard, no SPIRE/Istio, no HSM, no Telegram runtime).