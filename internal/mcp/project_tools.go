package mcp

import (
	"context"
	"errors"
	"strings"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func (s *Server) runnerTool(ctx context.Context, req core.RunnerToolRequest) (core.RunnerToolResponse, error) {
	host := strings.TrimSpace(s.engine.State().Preferences.OpenClawSSHHost)
	if host == "" {
		return core.RunnerToolResponse{}, errors.New("cluster project requires a configured Workbench Runner SSH host")
	}
	return core.RunRunnerToolSSH(ctx, host, req)
}

func (s *Server) listProjectFiles(ctx context.Context, project, subdir string, limit int) ([]string, error) {
	if core.IsRunnerProjectReference(project) {
		response, err := s.runnerTool(ctx, core.RunnerToolRequest{Action: "list_files", Project: project, Subdir: subdir, Limit: limit})
		return response.Files, err
	}
	return core.ListProjectFiles(project, subdir, limit)
}

func (s *Server) searchProjectText(ctx context.Context, project, query, subdir string, limit int) ([]core.SearchHit, error) {
	if core.IsRunnerProjectReference(project) {
		response, err := s.runnerTool(ctx, core.RunnerToolRequest{Action: "search_text", Project: project, Query: query, Subdir: subdir, Limit: limit})
		return response.Hits, err
	}
	return core.SearchProjectText(project, query, subdir, limit)
}

func (s *Server) readProjectFile(ctx context.Context, project, path string, startLine, endLine int) (string, error) {
	if core.IsRunnerProjectReference(project) {
		response, err := s.runnerTool(ctx, core.RunnerToolRequest{Action: "read_file", Project: project, Path: path, StartLine: startLine, EndLine: endLine})
		return response.Content, err
	}
	return core.ReadProjectFile(project, path, startLine, endLine)
}

func (s *Server) applyProjectPatch(ctx context.Context, project, patch string) (string, error) {
	if core.IsRunnerProjectReference(project) {
		response, err := s.runnerTool(ctx, core.RunnerToolRequest{Action: "apply_patch", Project: project, Patch: patch})
		return response.Output, err
	}
	return core.ApplyPatch(ctx, project, patch)
}

func (s *Server) runProjectSafeCommand(ctx context.Context, project, command string) (string, error) {
	if core.IsRunnerProjectReference(project) {
		response, err := s.runnerTool(ctx, core.RunnerToolRequest{Action: "run_safe_command", Project: project, Command: command})
		return response.Output, err
	}
	return core.RunSafeCommand(ctx, project, command)
}
