package handler

import (
	"context"

	streamingv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/streaming/v1"
	"github.com/nomados/nomados/services/streaming-service/internal/webrtc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StreamingServiceGRPCAdapter implements the gRPC StreamingServiceServer
// by delegating to the SignalingServer for WebRTC signaling operations.
type StreamingServiceGRPCAdapter struct {
	streamingv1.UnimplementedStreamingServiceServer
	signaling *webrtc.SignalingServer
}

// NewStreamingServiceGRPCAdapter creates a new gRPC adapter for the streaming service.
func NewStreamingServiceGRPCAdapter(signaling *webrtc.SignalingServer) *StreamingServiceGRPCAdapter {
	return &StreamingServiceGRPCAdapter{signaling: signaling}
}

// ProcessOffer handles a gRPC request to process a WebRTC SDP offer.
// It publishes the offer via NATS and waits for an answer from the browser runtime.
func (a *StreamingServiceGRPCAdapter) ProcessOffer(ctx context.Context, req *streamingv1.SignalingRequest) (*streamingv1.SignalingResponse, error) {
	if req.WorkspaceId == "" {
		return nil, status.Error(codes.InvalidArgument, "workspace_id is required")
	}
	if req.SdpOffer == "" {
		return nil, status.Error(codes.InvalidArgument, "sdp_offer is required")
	}

	answer, err := a.signaling.ProcessOffer(ctx, req.WorkspaceId, req.SdpOffer)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process offer: %v", err)
	}

	return &streamingv1.SignalingResponse{
		SdpAnswer:   answer.SDP,
		WorkspaceId: answer.WorkspaceID,
	}, nil
}

// ProcessICECandidate handles a gRPC request to relay an ICE candidate.
// It publishes the candidate via NATS for the browser runtime to consume.
func (a *StreamingServiceGRPCAdapter) ProcessICECandidate(ctx context.Context, req *streamingv1.ICECandidateRequest) (*streamingv1.ICECandidateResponse, error) {
	if req.WorkspaceId == "" {
		return nil, status.Error(codes.InvalidArgument, "workspace_id is required")
	}
	if req.Candidate == "" {
		return nil, status.Error(codes.InvalidArgument, "candidate is required")
	}

	err := a.signaling.ProcessICECandidate(ctx, req.WorkspaceId, req.Candidate, req.SdpMid, req.SdpMlineIndex)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process ICE candidate: %v", err)
	}

	return &streamingv1.ICECandidateResponse{
		Success: true,
	}, nil
}

// GetAnswer handles a gRPC request to retrieve a previously computed SDP answer
// for a workspace. It checks local cache first, then waits briefly on NATS.
func (a *StreamingServiceGRPCAdapter) GetAnswer(ctx context.Context, req *streamingv1.GetAnswerRequest) (*streamingv1.SignalingResponse, error) {
	if req.WorkspaceId == "" {
		return nil, status.Error(codes.InvalidArgument, "workspace_id is required")
	}

	answer, err := a.signaling.GetAnswer(ctx, req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "no answer found for workspace %s: %v", req.WorkspaceId, err)
	}

	return &streamingv1.SignalingResponse{
		SdpAnswer:   answer.SDP,
		WorkspaceId: answer.WorkspaceID,
	}, nil
}