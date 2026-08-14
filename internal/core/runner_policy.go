package core

import (
	"errors"
	"strings"
)

type RunnerPolicyRequest struct {
	Action    string `json:"action"`
	Project   string `json:"project"`
	RemoteURL string `json:"remote_url,omitempty"`
}

type RunnerPolicyResponse struct {
	OK         bool               `json:"ok"`
	Action     string             `json:"action,omitempty"`
	Project    string             `json:"project,omitempty"`
	Configured bool               `json:"configured,omitempty"`
	Policy     *PublicationPolicy `json:"policy,omitempty"`
	Deleted    bool               `json:"deleted,omitempty"`
	Error      string             `json:"error,omitempty"`
}

// ApplyRunnerPublicationPolicy is an operator control-plane action. It resolves
// project identity exactly like runner task execution, then reads or mutates
// the private local publication-policy store. It is deliberately not exposed
// through Workbench's model-safe command surface.
func ApplyRunnerPublicationPolicy(req RunnerPolicyRequest) (RunnerPolicyResponse, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	project := strings.TrimSpace(req.Project)
	if project == "" {
		return RunnerPolicyResponse{}, errors.New("publication policy project is required")
	}
	switch action {
	case "get", "prepare", "delete":
		if strings.TrimSpace(req.RemoteURL) != "" {
			return RunnerPolicyResponse{}, errors.New("remote URL is only valid for publish policy")
		}
	case "publish":
		if err := validatePublishRemote(strings.TrimSpace(req.RemoteURL)); err != nil {
			return RunnerPolicyResponse{}, err
		}
	default:
		return RunnerPolicyResponse{}, errors.New("publication policy action must be get, prepare, publish or delete")
	}

	resolved, err := ResolveRunnerProject(project)
	if err != nil {
		return RunnerPolicyResponse{}, err
	}
	response := RunnerPolicyResponse{OK: true, Action: action, Project: resolved}
	switch action {
	case "get":
		policy, configured, err := PublicationPolicyFor(resolved)
		if err != nil {
			return RunnerPolicyResponse{}, err
		}
		response.Configured = configured
		if configured {
			response.Policy = &policy
		}
	case "prepare":
		policy, err := SavePublicationPolicy(PublicationPolicy{Project: resolved, Mode: PublicationPrepare})
		if err != nil {
			return RunnerPolicyResponse{}, err
		}
		response.Policy = &policy
	case "publish":
		policy, err := SavePublicationPolicy(PublicationPolicy{Project: resolved, Mode: PublicationPublish, RemoteURL: strings.TrimSpace(req.RemoteURL)})
		if err != nil {
			return RunnerPolicyResponse{}, err
		}
		response.Policy = &policy
	case "delete":
		if err := DeletePublicationPolicy(resolved); err != nil {
			return RunnerPolicyResponse{}, err
		}
		response.Deleted = true
	}
	return response, nil
}
