package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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

// RunnerProviderInfo is deliberately privacy-minimal. The desktop may learn
// whether a known coding worker is usable on the execution host, but runner
// command paths, account identifiers, raw auth output and host filesystem
// details never cross this control channel.
type RunnerProviderInfo struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Capability    string    `json:"capability"`
	Status        string    `json:"status"`
	Cost          CostClass `json:"cost"`
	Installed     bool      `json:"installed"`
	Authenticated bool      `json:"authenticated"`
	Ready         bool      `json:"ready"`
}

type RunnerToolResponse struct {
	OK        bool                 `json:"ok"`
	Projects  []RunnerProjectInfo  `json:"projects,omitempty"`
	Providers []RunnerProviderInfo `json:"providers,omitempty"`
	Files     []string             `json:"files,omitempty"`
	Hits      []SearchHit          `json:"hits,omitempty"`
	Content   string               `json:"content,omitempty"`
	Output    string               `json:"output,omitempty"`
	Error     string               `json:"error,omitempty"`
}

func ApplyRunnerToolRequest(ctx context.Context, req RunnerToolRequest) (RunnerToolResponse, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case "list_projects":
		projects, err := listRunnerProjects(ctx)
		if err != nil {
			return RunnerToolResponse{Error: "runner project discovery is unavailable"}, err
		}
		return RunnerToolResponse{OK: true, Projects: projects}, nil
	case "list_providers":
		providers := providerInventoryWithConfiguredHarness(ScanProviders(), Preferences{})
		providers = ApplyProviderHealth(providers, time.Now())
		safe := safeRunnerProviderInventory(providers)
		// Model discovery is an optional refinement of the already-usable runner
		// inventory. If OpenClaw is missing or its catalogue cannot currently be
		// read, provider discovery still succeeds and ordinary routing is intact.
		if catalog, err := DiscoverOpenClawCloudModels(ctx, ""); err == nil {
			for _, model := range catalog.Models {
				if info, ok := runnerCloudModelProviderInfo(model); ok {
					safe = append(safe, info)
				}
			}
		}
		return RunnerToolResponse{OK: true, Providers: safe}, nil
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

func safeRunnerProviderInventory(providers []Provider) []RunnerProviderInfo {
	providers = append([]Provider(nil), providers...)
	sort.SliceStable(providers, func(i, j int) bool {
		if providers[i].Priority != providers[j].Priority {
			return providers[i].Priority < providers[j].Priority
		}
		return strings.ToLower(providers[i].Name) < strings.ToLower(providers[j].Name)
	})
	out := make([]RunnerProviderInfo, 0, len(providers))
	for _, provider := range providers {
		if !IsCodingWorkerProvider(provider) {
			continue
		}
		out = append(out, RunnerProviderInfo{
			ID:            provider.ID,
			Name:          provider.Name,
			Capability:    provider.Capability,
			Status:        strings.TrimSpace(provider.Status),
			Cost:          provider.Cost,
			Installed:     provider.Installed,
			Authenticated: provider.Authenticated,
			Ready:         ProviderReadyForCoding(provider),
		})
	}
	return out
}

func runnerCloudModelProviderInfo(model OpenClawCloudModel) (RunnerProviderInfo, bool) {
	id, err := RunnerCloudModelProviderID(model.Key)
	if err != nil || !model.Available {
		return RunnerProviderInfo{}, false
	}
	capability := "OpenClaw cloud model · " + canonicalOpenClawProvider(model.Provider)
	if strings.TrimSpace(model.Input) != "" {
		capability += " · " + strings.TrimSpace(model.Input)
	} else if model.Image {
		capability += " · text+image"
	} else {
		capability += " · text"
	}
	if model.ContextWindow > 0 {
		capability += fmt.Sprintf(" · context %dk", (model.ContextWindow+999)/1000)
	}
	if model.ContextTokens > 0 && model.ContextTokens != model.ContextWindow {
		capability += fmt.Sprintf(" · usable %dk", (model.ContextTokens+999)/1000)
	}

	status := "available · select and Connect to make OpenClaw default"
	ready := false
	if model.Default {
		status = "current OpenClaw default · routine cloud preference"
		ready = true
	}
	if model.Cooling {
		status += " · Workbench cooldown"
		if model.CooldownReason != "" {
			status += " (" + model.CooldownReason + ")"
		}
		ready = false
	}
	name := strings.TrimSpace(model.Name)
	if name == "" {
		_, name = splitOpenClawModelKey(model.Key)
	}
	return RunnerProviderInfo{
		ID:            id,
		Name:          "OpenClaw model · " + name,
		Capability:    capability,
		Status:        status,
		Cost:          CostIncluded,
		Installed:     true,
		Authenticated: true,
		Ready:         ready,
	}, true
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
