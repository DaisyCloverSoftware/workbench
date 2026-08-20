package core

import (
	"errors"
	"strings"
)

const (
	HostBridgeRPCPoll     = "poll"
	HostBridgeRPCComplete = "complete"
)

type HostBridgeRPCRequest struct {
	Action    string               `json:"action"`
	Heartbeat *HostBridgeHeartbeat `json:"heartbeat,omitempty"`
	HostID    string               `json:"host_id,omitempty"`
	JobID     string               `json:"job_id,omitempty"`
	Result    *HostJobResult       `json:"result,omitempty"`
	Error     string               `json:"error,omitempty"`
}

type HostBridgeRPCResponse struct {
	OK    bool            `json:"ok"`
	Host  *HostBridgeHost `json:"host,omitempty"`
	Job   *HostJob        `json:"job,omitempty"`
	Error string          `json:"error,omitempty"`
}

// ApplyHostBridgeRPC is the deliberately tiny machine-facing host protocol.
// It accepts typed data only: a Windows host can poll (heartbeat + claim at
// most one queued job) or complete a job it already claimed. There is no
// command, shell, executable, URL or path field in this protocol.
func ApplyHostBridgeRPC(req HostBridgeRPCRequest) (HostBridgeRPCResponse, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case HostBridgeRPCPoll:
		if req.Heartbeat == nil {
			return HostBridgeRPCResponse{}, errors.New("host bridge poll requires a heartbeat")
		}
		if strings.TrimSpace(req.HostID) != "" || strings.TrimSpace(req.JobID) != "" || req.Result != nil || strings.TrimSpace(req.Error) != "" {
			return HostBridgeRPCResponse{}, errors.New("host bridge poll contains completion-only fields")
		}
		host, err := RecordHostBridgeHeartbeat(*req.Heartbeat)
		if err != nil {
			return HostBridgeRPCResponse{}, err
		}
		job, err := ClaimHostBridgeJob(host.HostID)
		if err != nil {
			return HostBridgeRPCResponse{}, err
		}
		return HostBridgeRPCResponse{OK: true, Host: &host, Job: job}, nil

	case HostBridgeRPCComplete:
		if req.Heartbeat != nil {
			return HostBridgeRPCResponse{}, errors.New("host bridge completion cannot contain a heartbeat")
		}
		if strings.TrimSpace(req.HostID) == "" || strings.TrimSpace(req.JobID) == "" || req.Result == nil {
			return HostBridgeRPCResponse{}, errors.New("host bridge completion requires host_id, job_id and result")
		}
		job, err := CompleteHostBridgeJob(req.HostID, req.JobID, *req.Result, req.Error)
		if err != nil {
			return HostBridgeRPCResponse{}, err
		}
		return HostBridgeRPCResponse{OK: true, Job: &job}, nil

	default:
		return HostBridgeRPCResponse{}, errors.New("host bridge action must be poll or complete")
	}
}
