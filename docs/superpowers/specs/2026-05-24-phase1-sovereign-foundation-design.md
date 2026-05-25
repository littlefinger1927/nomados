# NomadOS Phase 1 — Sovereign Foundation Design

## Overview

NomadOS is a distributed zero-trust sovereign compute platform. Phase 1 delivers the core loop: **authenticate securely → create isolated workspace → operate inside it**. This phase focuses on the minimum viable architecture that demonstrates the platform's core value proposition while establishing the service boundaries and security model for future phases.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scope | Core loop only (auth → workspace → operate) | Prove the model works before building observability, advanced networking, vault |
| Language | Go backend, Rust for Tauri client + crypto | Simpler team skillset, Rust where safety matters most |
| Architecture | Monorepo microservices with gRPC over mTLS | Matches spec structure, enforces boundaries, clean migration path to SPIRE/Istio |
| Infra | Single-node Docker Compose | Minimize ops overhead, scale later |
| Auth | Custom auth service with WebAuthn/passkeys | Full control, matches spec's sovereignty-first principle |
| Workspace isolation | Docker containers (Firecracker in Phase 2) | Simpler dev experience, adequate isolation for Phase 1 |
| Encryption | App-level XChaCha20-Poly1305 per-workspace keys | Works with any backend, granular key rotation |
| Service mesh | Direct mTLS (no SPIRE/Istio until Phase 5) | Right-sized for single-node, cert rotation via auth-service |
| Browser | Remote Chromium streaming via WebRTC/TURN | Matches spec's streaming model, full anti-fingerprinting control |

## Services

| Service | Language | Responsibility |
|---------|----------|---------------|
| `gateway-service` | Go | TLS termination, rate limiting, request routing, WAF baseline |
| `auth-service` | Go | WebAuthn/passkey registration, login, MFA, device attestation |
| `session-service` | Go | Session lifecycle, rotation, device binding, revocation |
| `workspace-orchestrator` | Go | Create/destroy/pause workspace containers, manage lifecycle |
| `browser-manager` | Go | Launch Chromium per workspace, apply anti-fingerprint profiles |
| `streaming-service` | Go | TURN relay, WebRTC video/audio stream management |
| `file-service` | Go | Encrypted file storage, upload/download, versioning |
| `vault-service` | Go | Key management, workspace key derivation, signing gateway stub |
| `observability` | Go | Structured log aggregation, basic metrics |

## Shared Packages

| Package | Language | Responsibility |
|---------|----------|---------------|
| `shared-types` | Go + TypeScript | Proto/gRPC definitions, shared types |
| `crypto` | Rust (shared library) | XChaCha20-Poly1305, Argon2id, HKDF, key derivation. Called directly from Tauri client (Rust). Go services call via CGo. |
| `auth-sdk` | Go | Token validation, mTLS helpers, certificate management |
| `logging` | Go | Structured logging, audit stream formatting |
| `ui-components` | TypeScript/React | Shared UI component library |

## Authentication & Session Design

### Registration Flow

1. User opens Tauri client → generates ed25519 device keypair
2. Submits username + WebAuthn/passkey credential
3. Auth service stores: user record, public key, device attestation hash
4. Returns initial session token (short-lived, bound to device key)

### Login Flow

1. User opens Tauri client → presents device key
2. Completes WebAuthn challenge (passkey/security key)
3. Auth service verifies credential + device attestation
4. Session service creates session: bound to device key, IP, timestamp
5. Returns session token (JWT signed by auth service, contains session ID + device ID)
6. Session token refreshes every 15 minutes via mTLS channel

### Session Hardening

- Every session token bound to device keypair — stolen tokens alone are useless
- Continuous rotation: refresh tokens every 15 min, access tokens every 5 min
- Risk scoring: new device, new IP, impossible travel → escalate to re-auth
- Remote revocation: session invalidation propagates via NATS
- No bearer JWT trust — tokens must be validated against session service

### Auth Database Schema

- `users` — id, username, created_at, status
- `devices` — id, user_id, public_key, attestation, trusted, last_seen
- `sessions` — id, user_id, device_id, created_at, expires_at, risk_score, ip_hash
- `mfa_tokens` — id, user_id, type, verified_at
- `recovery_keys` — id, user_id, encrypted_key, used_at
- `audit_logs` — id, actor_id, action, target, timestamp, signature

## Workspace Lifecycle & Browser Streaming

### Workspace Lifecycle

1. User creates workspace → orchestrator provisions Docker container with:
   - Isolated network namespace (Docker network)
   - Chromium instance with anti-fingerprint profile
   - Dedicated filesystem with per-workspace encryption key
   - Workspace agent (Go) for orchestration commands
2. Browser manager launches Chromium with patched flags, streams viewport via WebRTC
3. Streaming service relays through TURN server (no direct peer exposure)
4. User operates browser remotely — all rendering happens server-side

### Workspace States

- `Creating` — container starting, Chromium launching
- `Running` — user is active, stream is live
- `Paused` — container frozen (cgroups freezer), state preserved
- `Stopping` — graceful shutdown, encrypting workspace data
- `Stopped` — container removed, encrypted state persisted to MinIO

### Browser Anti-Fingerprinting (Phase 1 Baseline)

- Deterministic viewport size
- Spoofed User-Agent
- Disabled WebRTC, QUIC, DNS prefetch
- Canvas/WebGL noise injection
- Timezone spoofing per workspace
- Font virtualization baseline

### Streaming Protocol

- Video: VP8/VP9 via WebRTC through TURN relay
- Input: encrypted WebRTC data channel for mouse/keyboard events
- No direct peer connectivity — TURN-only as spec requires
- Traffic padding and bitrate normalization deferred to Phase 2

## Encrypted Storage & Key Management

### Encryption Model

- **Master key**: derived from user password via Argon2id (never stored plaintext)
- **Per-workspace key**: derived from master key using HKDF with workspace ID as salt
- **Per-file key**: derived from workspace key using HKDF with file ID as salt
- **Algorithm**: XChaCha20-Poly1305 (AEAD) for all encryption
- **Key derivation**: Rust crypto shared library — called directly by Tauri client, via CGo by Go services

### Storage Tiers

| Tier | Storage | Contents |
|------|---------|----------|
| Tier 1 | PostgreSQL | Metadata (filenames, sizes, versions, permissions) |
| Tier 2 | Redis | Decrypted workspace cache (ephemeral, memory-only) |
| Tier 3 | MinIO | Encrypted file objects |
| Tier 4 | Encrypted volume | Workspace filesystem snapshots |

### File Upload Flow

1. Client encrypts file with per-file key (client-side encryption)
2. Encrypted blob uploaded to file-service
3. File-service stores blob in MinIO, metadata in PostgreSQL
4. Decryption only happens client-side or inside workspace container

### Key Rotation

- Workspace keys rotated by re-encrypting file keys
- Master key rotation requires re-encrypting workspace keys
- Rotation is async — old keys stay valid until rotation completes

### Vault Service

- Stores encrypted master keys (never plaintext in memory longer than needed)
- Signs workspace key derivation requests
- Phase 1 stub for signing gateway (full implementation in Phase 6)
- Memory zeroization after key operations

## Tauri Client

### Architecture

```
Tauri (Rust)
├── Auth Module — WebAuthn, device key management, session handling
├── Workspace Module — list, create, pause, destroy workspaces
├── Stream Module — WebRTC viewer, input relay
├── Crypto Module — local encryption/decryption, key derivation
├── Network Module — gateway mTLS connection, kill switch (WireGuard in Phase 3)
└── UI Layer — Next.js web app loaded in Tauri webview
```

### Client Responsibilities

- Login/registration UI (WebAuthn/passkey flow)
- Workspace list and management
- Browser stream display (WebRTC viewer)
- Local encrypted cache (workspace metadata, settings)
- Secure tunnel management for workspace traffic (mTLS to gateway; WireGuard in Phase 3)
- Biometric unlock (macOS Keychain, Windows Credential Manager, Linux Secret Service)

## Networking

### Gateway Routing

```
Client → mTLS → Gateway
                  ├── /auth/*     → auth-service
                  ├── /session/*  → session-service
                  ├── /workspace/* → workspace-orchestrator
                  ├── /stream/*   → streaming-service
                  ├── /file/*     → file-service
                  └── /vault/*    → vault-service
```

### Inter-Service Communication

- gRPC over mTLS for synchronous service-to-service calls
- NATS for async events (session revocation, workspace state changes, audit logs)
- Certificate rotation via auth-service (replaces SPIRE for Phase 1)

### Workspace Network Isolation

- Each workspace gets its own Docker network
- DNS handled by isolated Unbound resolver per workspace
- All workspace traffic routed through gateway (no direct internet access)
- Kill switch on gateway disconnect

## Monorepo Structure

```
nomados/
├── apps/
│   ├── web-desktop/          # Next.js + Tauri web UI
│   ├── auth-portal/          # Standalone auth web UI (recovery, admin)
│   └── tauri-client/         # Rust Tauri shell (macOS, Windows, Linux)
│
├── services/
│   ├── gateway-service/      # Go — TLS, routing, rate limiting
│   ├── auth-service/          # Go — WebAuthn, MFA, device trust
│   ├── session-service/      # Go — session lifecycle, rotation
│   ├── workspace-orchestrator/ # Go — container lifecycle
│   ├── browser-manager/      # Go — Chromium launch, anti-fingerprint
│   ├── streaming-service/    # Go — WebRTC TURN relay
│   ├── file-service/         # Go — encrypted storage
│   ├── vault-service/        # Go — key management, signing stub
│   └── observability/        # Go — structured log aggregation
│
├── infrastructure/
│   ├── docker/               # Dockerfiles, Compose configs
│   ├── wireguard/            # WireGuard config templates
│   └── ci-cd/                # GitHub Actions workflows
│
├── packages/
│   ├── shared-types/         # Proto/gRPC definitions, shared Go+TS types
│   ├── crypto/               # Rust — XChaCha20, Argon2id, HKDF
│   ├── auth-sdk/             # Go — token validation, mTLS helpers
│   ├── logging/              # Go — structured logging, audit stream
│   └── ui-components/        # React component library
│
├── security/
│   ├── policies/             # Network policies, rate limits
│   └── threat-models/        # Phase 1 threat model docs
│
├── docs/
│   ├── architecture/
│   ├── deployment/
│   ├── APIs/
│   └── security/
│
└── scripts/
    ├── dev.sh                # Start all services
    ├── test.sh               # Run all tests
    └── build.sh              # Build all binaries
```

## Dev Environment

- Docker Compose runs all services locally
- `scripts/dev.sh` starts: PostgreSQL, Redis, MinIO, NATS, all Go services, TURN server
- Tauri client connects to `localhost` gateway
- Hot reload for Go services (air), Next.js (next dev)
- Proto code generation via `buf`

## Testing Requirements (Phase 1)

- Unit tests for all services
- Integration tests for auth flow (registration → login → session)
- Integration tests for workspace lifecycle (create → stream → pause → resume → destroy)
- Isolation tests (verify workspace network separation)
- Encryption tests (verify key derivation, encryption/decryption)
- Security tests (verify mTLS, token validation, session binding)

## Phase Dependencies

- Phase 2 (Hardened Workspaces) will add Firecracker MicroVMs replacing Docker containers
- Phase 3 (Network Fabric) will add WireGuard orchestration and proxy chaining
- Phase 5 (Security Hardening) will add SPIRE/Istio service mesh replacing direct mTLS
- Phase 6 (Vault + Signing) will expand vault-service from stub to full implementation

## Success Criteria

Phase 1 succeeds when a user can:

1. Register an account with WebAuthn/passkey from the Tauri client
2. Log in from any device with phishing-resistant authentication
3. Create an isolated workspace with its own browser runtime
4. Operate the browser remotely through encrypted streaming
5. Store and retrieve encrypted files within workspaces
6. Pause/resume workspaces with state preservation
7. Destroy workspaces with encrypted state cleanup
8. Have all inter-service communication over mTLS
9. Have all workspace traffic isolated from other workspaces