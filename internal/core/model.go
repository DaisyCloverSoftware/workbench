package core

import "time"

type CostClass string

const (
	CostZero     CostClass = "zero-marginal"
	CostIncluded CostClass = "included-subscription"
	CostScarce   CostClass = "scarce-agentic"
	CostMetered  CostClass = "metered-api"
)

type Provider struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Capability    string    `json:"capability"`
	Command       string    `json:"command,omitempty"`
	Installed     bool      `json:"installed"`
	Authenticated bool      `json:"authenticated"`
	Status        string    `json:"status"`
	Cost          CostClass `json:"cost"`
	Priority      int       `json:"priority"`
	CanWrite      bool      `json:"can_write"`
	CanRunTools   bool      `json:"can_run_tools"`
	Notes         string    `json:"notes,omitempty"`
}

type TaskStatus string

const (
	TaskQueued            TaskStatus = "queued"
	TaskRouting           TaskStatus = "routing"
	TaskRunning           TaskStatus = "running"
	TaskWaitingRetry      TaskStatus = "waiting_retry"
	TaskWaitingDependency TaskStatus = "waiting_dependency"
	TaskNeedsAttention    TaskStatus = "needs_attention"
	TaskCompleted         TaskStatus = "completed"
	TaskFailed            TaskStatus = "failed"
	TaskCancelled         TaskStatus = "cancelled"
)

type TaskMode string

const (
	// TaskModeDevelopment is the historic/default Workbench lane. ChatGPT owns
	// the reasoning and source changes; autonomous coding workers are only an
	// escalation path when safe hands are insufficient.
	TaskModeDevelopment TaskMode = "development"
	// TaskModeOperations is deliberately separate. It is for the host/cluster /
	// runtime actions the user otherwise has to copy into OpenClaw manually.
	// OpenClaw acts as an operator in this lane and is not allowed to become the
	// application coder.
	TaskModeOperations TaskMode = "operations"
)

func IsOperationsTask(task Task) bool {
	return task.Mode == TaskModeOperations
}

type DependencyKind string

const DependencyGitHubActions DependencyKind = "github_actions"

// TaskDependency is a Workbench-owned external wait. It deliberately stores
// only the minimum durable locator/state required to resume work without
// keeping an AI worker alive. Secrets and raw provider/API responses do not
// belong here.
type TaskDependency struct {
	Kind          DependencyKind `json:"kind"`
	Reason        string         `json:"reason,omitempty"`
	Repository    string         `json:"repository,omitempty"`
	RunID         int64          `json:"run_id,omitempty"`
	State         string         `json:"state,omitempty"`
	Conclusion    string         `json:"conclusion,omitempty"`
	CheckCount    int            `json:"check_count,omitempty"`
	StartedAt     time.Time      `json:"started_at"`
	LastCheckedAt time.Time      `json:"last_checked_at,omitempty"`
	NextCheckAt   time.Time      `json:"next_check_at"`
}

type Task struct {
	ID                 string            `json:"id"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	Origin             string            `json:"origin"`
	Title              string            `json:"title"`
	Intent             string            `json:"intent"`
	ProjectPath        string            `json:"project_path"`
	Mode               TaskMode          `json:"mode,omitempty"`
	Status             TaskStatus        `json:"status"`
	Archived           bool              `json:"archived,omitempty"`
	ProviderID         string            `json:"provider_id,omitempty"`
	CloudModelOverride string            `json:"cloud_model_override,omitempty"`
	RouteReason        string            `json:"route_reason,omitempty"`
	Output             string            `json:"output,omitempty"`
	Error              string            `json:"error,omitempty"`
	AttentionQuestion  string            `json:"attention_question,omitempty"`
	HumanAnswer        string            `json:"human_answer,omitempty"`
	Attempts           []string          `json:"attempts,omitempty"`
	Review             *TaskReviewResult `json:"review,omitempty"`
	ConsumesWork       bool              `json:"consumes_work"`
	AutoRetryCount     int               `json:"auto_retry_count,omitempty"`
	RetryAt            *time.Time        `json:"retry_at,omitempty"`
	Dependency         *TaskDependency   `json:"dependency,omitempty"`
	DependencyResult   string            `json:"dependency_result,omitempty"`
	StartedAt          *time.Time        `json:"started_at,omitempty"`
	FinishedAt         *time.Time        `json:"finished_at,omitempty"`

	// memoryProjectPath is transient execution metadata. Isolated worker copies
	// keep ProjectPath pointed at the task worktree while memory lookup/capture
	// remains scoped to the real source project. It is intentionally unexported
	// so it is never persisted, relayed, or exposed to model-facing JSON.
	memoryProjectPath string
}

// Project is a local Workbench workspace entry. ProjectPath/Notes remain on
// State as a compatibility mirror for the currently selected project while the
// product migrates from its original single-project state shape.
type Project struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Notes      string    `json:"notes,omitempty"`
	Pinned     bool      `json:"pinned,omitempty"`
	AddedAt    time.Time `json:"added_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}

type Preferences struct {
	AvoidWorkUsage      bool   `json:"avoid_work_usage"`
	AllowMeteredAPI     bool   `json:"allow_metered_api"`
	AutonomyMode        string `json:"autonomy_mode"`
	OpenClawSSHHost     string `json:"openclaw_ssh_host,omitempty"`
	HarnessAdapterPath  string `json:"harness_adapter_path,omitempty"`
	OpenClawCommand     string `json:"openclaw_command,omitempty"`
	NotificationCommand string `json:"notification_command,omitempty"`
	MCPPort             int    `json:"mcp_port"`
	MCPToken            string `json:"mcp_token"`
}

type SecretRef struct {
	Name       string    `json:"name"`
	Ciphertext string    `json:"ciphertext"`
	CreatedAt  time.Time `json:"created_at"`
}

type State struct {
	Version         int         `json:"version"`
	Projects        []Project   `json:"projects,omitempty"`
	ActiveProjectID string      `json:"active_project_id,omitempty"`
	ProjectPath     string      `json:"project_path"`
	Notes           string      `json:"notes"`
	Tasks           []Task      `json:"tasks"`
	Secrets         []SecretRef `json:"secrets"`
	Preferences     Preferences `json:"preferences"`
}

func DefaultState() State {
	return State{
		Version: 3,
		Preferences: Preferences{
			AvoidWorkUsage:  true,
			AllowMeteredAPI: false,
			AutonomyMode:    "trusted-repo",
			MCPPort:         8765,
		},
	}
}
