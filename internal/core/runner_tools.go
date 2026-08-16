package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type RunnerToolRequest struct {
	Action    string `json:"action"`
	Project   string `json:"project,omitempty"`
	Path      string `json:"path,omitempty"`
	Subdir    string `json:"subdir,omitempty"`
	Query     string `json:"query,omitempty"`
	Patch     string `json:"patch,omitempty"`
	Command   string `json:"command,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type RunnerProjectInfo struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
}

type RunnerToolResponse struct {
	OK       bool                `json:"ok"`
	Projects []RunnerProjectInfo `json:"projects,omitempty"`
	Files    []string            `json:"files,omitempty"`
	Hits     []SearchHit         `json:"hits,omitempty"`
	Content  string              `json:"content,omitempty"`
	Output   string              `json:"output,omitempty"`
	Error    string              `json:"error,omitempty"`
}

func ApplyRunnerToolRequest(ctx context.Context, req RunnerToolRequest) (RunnerToolResponse, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "list_projects" {
		projects, err := listRunnerProjects(ctx)
		if err != nil {
			return RunnerToolResponse{Error: "runner project discovery is unavailable"}, err
		}
		return RunnerToolResponse{OK: true, Projects: projects}, nil
	}

	if strings.TrimSpace(req.Project) == "" {
		return RunnerToolResponse{Error: "runner project is required"}, errors.New("runner project is required")
	}
	project, err := ResolveRunnerProject(req.Project)
	if err != nil {
		return RunnerToolResponse{Error: "runner project is unavailable"}, err
	}

	response := RunnerToolResponse{OK: true}
	switch action {
	case "list_files":
		response.Files, err = ListProjectFiles(project, req.Subdir, req.Limit)
	case "search_text":
		response.Hits, err = SearchProjectText(project, req.Query, req.Subdir, req.Limit)
	case "read_file":
		response.Content, err = ReadProjectFile(project, req.Path, req.StartLine, req.EndLine)
	case "apply_patch":
		response.Output, err = ApplyPatch(ctx, project, req.Patch)
	case "run_safe_command":
		response.Output, err = RunSafeCommand(ctx, project, req.Command)
	default:
		return RunnerToolResponse{Error: "unsupported runner tool action"}, errors.New("unsupported runner tool action")
	}
	if err != nil {
		return RunnerToolResponse{Error: "runner tool operation failed"}, err
	}
	return response, nil
}

func listRunnerProjects(ctx context.Context) ([]RunnerProjectInfo, error) {
	configuredRoot, err := runnerRoot()
	if err != nil {
		return nil, err
	}
	root, err := canonicalRunnerDirectory(configuredRoot)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	projects := make([]RunnerProjectInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || len(projects) >= 500 {
			continue
		}
		candidate := filepath.Join(root, entry.Name())
		resolved, resolveErr := canonicalRunnerDirectory(candidate)
		if resolveErr != nil || !withinRoot(root, resolved) {
			continue
		}
		gitRoot, gitErr := runGitLimited(ctx, resolved, 4096, "rev-parse", "--show-toplevel")
		if gitErr != nil {
			continue
		}
		canonicalGitRoot, canonicalErr := canonicalRunnerDirectory(strings.TrimSpace(gitRoot))
		if canonicalErr != nil || filepath.Clean(canonicalGitRoot) != filepath.Clean(resolved) {
			continue
		}
		ref, refErr := RunnerProjectReference(entry.Name())
		if refErr != nil {
			continue
		}
		projects = append(projects, RunnerProjectInfo{Name: entry.Name(), Ref: ref})
	}
	sort.Slice(projects, func(i, j int) bool {
		return strings.ToLower(projects[i].Name) < strings.ToLower(projects[j].Name)
	})
	return projects, nil
}
