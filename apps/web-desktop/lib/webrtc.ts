/**
 * WebRTC peer connection client for NomadOS workspace streaming.
 *
 * Manages the browser-side WebRTC connection that streams a remote
 * workspace viewport. The signaling flow is:
 *   1. Create RTCPeerConnection + data channel
 *   2. Create and set local SDP offer
 *   3. Send offer to the gRPC signaling service via the gateway
 *   4. Receive and set the remote SDP answer
 *   5. Exchange ICE candidates through the signaling service
 *
 * The data channel (`input`) is created but not wired for input relay
 * in this phase. Phase 6 will add input event relay.
 */

import { apiPost } from './auth';

export interface ConnectionConfig {
  workspaceId: string;
  onStateChange?: (state: ConnectionState) => void;
  onDataChannel?: (channel: RTCDataChannel) => void;
  maxReconnectAttempts?: number;
}

export type ConnectionState =
  | 'disconnected'
  | 'connecting'
  | 'connected'
  | 'failed'
  | 'reconnecting';

export class WorkspaceConnection {
  private peerConnection: RTCPeerConnection | null = null;
  private dataChannel: RTCDataChannel | null = null;
  private config: ConnectionConfig;
  private _state: ConnectionState = 'disconnected';
  private reconnectAttempts = 0;
  private maxReconnectAttempts: number;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  constructor(config: ConnectionConfig) {
    this.config = config;
    this.maxReconnectAttempts = config.maxReconnectAttempts ?? 5;
  }

  get state(): ConnectionState {
    return this._state;
  }

  private setState(state: ConnectionState): void {
    this._state = state;
    this.config.onStateChange?.(state);
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.setState('failed');
      return;
    }

    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts++;
    this.setState('reconnecting');

    this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, delay);
  }

  async connect(): Promise<void> {
    this.setState('connecting');

    try {
      // Create peer connection with TURN server config.
      // Credentials match infrastructure/docker/turn/turnserver.conf
      this.peerConnection = new RTCPeerConnection({
        iceServers: [
          {
            urls: 'turn:localhost:3478',
            username: 'nomados',
            credential: 'nomados_dev_secret',
          },
          {
            urls: 'stun:localhost:3478',
          },
        ],
      });

      // Create data channel for input relay (Phase 5: create but don't wire yet)
      this.dataChannel = this.peerConnection.createDataChannel('input', {
        ordered: true,
      });
      this.dataChannel.onopen = () => {
        this.reconnectAttempts = 0;
        this.setState('connected');
      };

      // Handle ICE candidates
      this.peerConnection.onicecandidate = async (event) => {
        if (event.candidate) {
          await this.sendICECandidate(event.candidate);
        }
      };

      // Handle connection state changes
      this.peerConnection.onconnectionstatechange = () => {
        switch (this.peerConnection?.connectionState) {
          case 'connected':
            this.reconnectAttempts = 0;
            this.setState('connected');
            break;
          case 'disconnected':
          case 'closed':
            this.scheduleReconnect();
            break;
          case 'failed':
            this.scheduleReconnect();
            break;
        }
      };

      // Create offer
      const offer = await this.peerConnection.createOffer();
      await this.peerConnection.setLocalDescription(offer);

      // Send offer to signaling service
      const answer = await this.sendOffer(offer);
      if (answer) {
        await this.peerConnection.setRemoteDescription(
          new RTCSessionDescription(answer),
        );
      }
      // If no answer yet, the connection will remain in "connecting" state.
      // In production, we'd use WebSocket or polling to wait for the answer.
    } catch (error) {
      this.setState('failed');
      throw error;
    }
  }

  private async sendOffer(
    offer: RTCSessionDescriptionInit,
  ): Promise<RTCSessionDescriptionInit | null> {
    try {
      const response = await apiPost('/v1/streaming/process_offer', {
        workspace_id: this.config.workspaceId,
        sdp_offer: offer.sdp,
      });

      if (!response.ok) {
        console.warn('Signaling offer failed, will retry:', response.status);
        return null;
      }

      const data = await response.json();
      if (data.sdp_answer) {
        return {
          type: 'answer' as RTCSdpType,
          sdp: data.sdp_answer,
        };
      }
      return null;
    } catch (error) {
      console.warn('Signaling offer error:', error);
      return null;
    }
  }

  private async sendICECandidate(candidate: RTCIceCandidate): Promise<void> {
    try {
      await apiPost('/v1/streaming/process_ice_candidate', {
        workspace_id: this.config.workspaceId,
        candidate: candidate.candidate,
        sdp_mid: candidate.sdpMid,
        sdp_mline_index: candidate.sdpMLineIndex,
      });
    } catch (error) {
      // ICE candidate sending is best-effort
      console.warn('Failed to send ICE candidate:', error);
    }
  }

  disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.reconnectAttempts = 0;
    if (this.dataChannel) {
      this.dataChannel.close();
      this.dataChannel = null;
    }
    if (this.peerConnection) {
      this.peerConnection.close();
      this.peerConnection = null;
    }
    this.setState('disconnected');
  }

  get dataChannelReady(): boolean {
    return this.dataChannel?.readyState === 'open';
  }

  sendData(data: string | ArrayBuffer): void {
    if (this.dataChannel?.readyState === 'open') {
      if (typeof data === 'string') {
        this.dataChannel.send(data);
      } else {
        this.dataChannel.send(new Uint8Array(data));
      }
    }
  }
}

/**
 * Check if WebRTC is available in the current browser.
 * Returns false during SSR (Next.js server-side render).
 */
export function isWebRTCAvailable(): boolean {
  return (
    typeof window !== 'undefined' &&
    !!window.RTCPeerConnection &&
    !!navigator.mediaDevices
  );
}