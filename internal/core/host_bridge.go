package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	HostBridgePlatformWindows = "windows"
	HostBridgeToolBlender     = "blender"
	HostBridgeToolUnreal      = "unreal"
	HostBridgeOperationVersion = "version"
)

var hostBridgeIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{7,95}$`)

type HostCapability struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
}

type HostBridgeHeartbeat struct {
	HostID       string                    `json:"host_id"`
	Label        string                    `json:"label"`
	Platform     string                    `json:"platform"`
	Arch         string                    `json:"arch"`
	Capabilities map[string]HostCapability `json:"capabilities,omitempty"`
}

type HostBridgeHost struct {
	HostID       string                    `json:"host_id"`
	Label        string                    `json:"label"`
	Platform     string                    `json:"platform"`
	Arch         string                    `json:"arch"`
	Capabilities map[string]HostCapability `json:"capabilities,omitempty"`
	LastSeen     string                    `json:"last_seen"`
	Online       bool                      `json:"online"`
}

type HostJobSpec struct {
	Tool      string `json:"tool"`
	Operation string `json:"operation"`
}

type HostJobResult struct {
	Output   string `json:"output,omitempty"`
	ExitCode int    `json:"exit_code"`
}

type HostJob struct {
	ID             string         `json:"id"`
	HostID         string         `json:"host_id"`
	Spec           HostJobSpec    `json:"spec"`
	Status         string         `json:"status"`
	ClaimedBy      string         `json:"claimed_by,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
	ClaimedAt      string         `json:"claimed_at,omitempty"`
	ClaimExpiresAt string         `json:"claim_expires_at,omitempty"`
	Result         *HostJobResult `json:"result,omitempty"`
	Error          string         `json:"error,omitempty"`
}

func hostBridgeRoot() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("WORKBENCH_HOST_BRIDGE_STATE_DIR")); configured != "" {
		return filepath.Abs(configured)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Workbench", "host-bridge"), nil
}

func withHostBridgeLock(fn func(root string) error) error {
	root, err := hostBridgeRoot()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "hosts"), 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "jobs"), 0o700); err != nil {
		return err
	}
	lock := filepath.Join(root, ".lock")
	deadline := time.Now().Add(3 * time.Second)
	for {
		err = os.Mkdir(lock, 0o700)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		if st, statErr := os.Stat(lock); statErr == nil && time.Since(st.ModTime()) > 30*time.Second {
			_ = os.Remove(lock)
			continue
		}
		if time.Now().After(deadline) {
			return errors.New("host bridge state is busy")
		}
		time.Sleep(20 * time.Millisecond)
	}
	defer os.Remove(lock)
	return fn(root)
}

func writeHostBridgeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".host-bridge-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readHostBridgeJSON(path string, dst any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(b) > 64<<10 {
		return errors.New("host bridge state file is oversized")
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func validateHostBridgeID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if !hostBridgeIDPattern.MatchString(id) {
		return "", errors.New("host bridge id is invalid")
	}
	return id, nil
}

func validateHostJobSpec(spec HostJobSpec) (HostJobSpec, error) {
	spec.Tool = strings.ToLower(strings.TrimSpace(spec.Tool))
	spec.Operation = strings.ToLower(strings.TrimSpace(spec.Operation))
	if spec.Operation != HostBridgeOperationVersion {
		return HostJobSpec{}, errors.New("host job operation is not allowlisted")
	}
	switch spec.Tool {
	case HostBridgeToolBlender, HostBridgeToolUnreal:
		return spec, nil
	default:
		return HostJobSpec{}, errors.New("host job tool is not allowlisted")
	}
}

func sanitizeHostHeartbeat(h HostBridgeHeartbeat) (HostBridgeHeartbeat, error) {
	id, err := validateHostBridgeID(h.HostID)
	if err != nil {
		return HostBridgeHeartbeat{}, err
	}
	h.HostID = id
	h.Label = strings.TrimSpace(h.Label)
	if h.Label == "" || len(h.Label) > 100 || LooksSecret(h.Label) {
		return HostBridgeHeartbeat{}, errors.New("host label is invalid")
	}
	h.Platform = strings.ToLower(strings.TrimSpace(h.Platform))
	if h.Platform != HostBridgePlatformWindows {
		return HostBridgeHeartbeat{}, errors.New("host platform is not supported")
	}
	h.Arch = strings.ToLower(strings.TrimSpace(h.Arch))
	if h.Arch == "" || len(h.Arch) > 32 {
		return HostBridgeHeartbeat{}, errors.New("host architecture is invalid")
	}
	clean := map[string]HostCapability{}
	for _, name := range []string{HostBridgeToolBlender, HostBridgeToolUnreal} {
		capability, ok := h.Capabilities[name]
		if !ok {
			continue
		}
		capability.Version = strings.TrimSpace(capability.Version)
		if len(capability.Version) > 256 || LooksSecret(capability.Version) {
			return HostBridgeHeartbeat{}, errors.New("host capability version is invalid")
		}
		if !capability.Installed {
			capability.Version = ""
		}
		clean[name] = capability
	}
	h.Capabilities = clean
	return h, nil
}

func RecordHostBridgeHeartbeat(h HostBridgeHeartbeat) (HostBridgeHost, error) {
	h, err := sanitizeHostHeartbeat(h)
	if err != nil {
		return HostBridgeHost{}, err
	}
	now := time.Now().UTC()
	host := HostBridgeHost{
		HostID: h.HostID, Label: h.Label, Platform: h.Platform, Arch: h.Arch,
		Capabilities: h.Capabilities, LastSeen: now.Format(time.RFC3339Nano), Online: true,
	}
	err = withHostBridgeLock(func(root string) error {
		return writeHostBridgeJSON(filepath.Join(root, "hosts", host.HostID+".json"), host)
	})
	return host, err
}

func ListHostBridgeHosts() ([]HostBridgeHost, error) {
	var hosts []HostBridgeHost
	err := withHostBridgeLock(func(root string) error {
		entries, err := os.ReadDir(filepath.Join(root, "hosts"))
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			var host HostBridgeHost
			if err := readHostBridgeJSON(filepath.Join(root, "hosts", entry.Name()), &host); err != nil {
				continue
			}
			seen, err := time.Parse(time.RFC3339Nano, host.LastSeen)
			if err != nil {
				continue
			}
			host.Online = now.Sub(seen) <= 2*time.Minute
			hosts = append(hosts, host)
		}
		return nil
	})
	sort.Slice(hosts, func(i, j int) bool {
		if hosts[i].Online != hosts[j].Online {
			return hosts[i].Online
		}
		return hosts[i].HostID < hosts[j].HostID
	})
	return hosts, err
}

func newHostJobID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "hostjob_" + hex.EncodeToString(b), nil
}

func SubmitHostBridgeJob(hostID string, spec HostJobSpec) (HostJob, error) {
	hostID, err := validateHostBridgeID(hostID)
	if err != nil {
		return HostJob{}, err
	}
	spec, err = validateHostJobSpec(spec)
	if err != nil {
		return HostJob{}, err
	}
	jobID, err := newHostJobID()
	if err != nil {
		return HostJob{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	job := HostJob{ID: jobID, HostID: hostID, Spec: spec, Status: "queued", CreatedAt: now, UpdatedAt: now}
	err = withHostBridgeLock(func(root string) error {
		var host HostBridgeHost
		if err := readHostBridgeJSON(filepath.Join(root, "hosts", hostID+".json"), &host); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return errors.New("target host is not registered")
			}
			return err
		}
		return writeHostBridgeJSON(filepath.Join(root, "jobs", job.ID+".json"), job)
	})
	return job, err
}

func GetHostBridgeJob(jobID string) (HostJob, error) {
	jobID, err := validateHostBridgeID(jobID)
	if err != nil {
		return HostJob{}, err
	}
	var job HostJob
	err = withHostBridgeLock(func(root string) error {
		return readHostBridgeJSON(filepath.Join(root, "jobs", jobID+".json"), &job)
	})
	if errors.Is(err, os.ErrNotExist) {
		return HostJob{}, errors.New("host job not found")
	}
	return job, err
}

func ClaimHostBridgeJob(hostID string) (*HostJob, error) {
	hostID, err := validateHostBridgeID(hostID)
	if err != nil {
		return nil, err
	}
	var claimed *HostJob
	err = withHostBridgeLock(func(root string) error {
		entries, err := os.ReadDir(filepath.Join(root, "jobs"))
		if err != nil {
			return err
		}
		jobs := make([]HostJob, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			var job HostJob
			if err := readHostBridgeJSON(filepath.Join(root, "jobs", entry.Name()), &job); err != nil || job.HostID != hostID {
				continue
			}
			jobs = append(jobs, job)
		}
		sort.Slice(jobs, func(i, j int) bool { return jobs[i].CreatedAt < jobs[j].CreatedAt })
		now := time.Now().UTC()
		for _, job := range jobs {
			eligible := job.Status == "queued"
			if job.Status == "claimed" && job.ClaimExpiresAt != "" {
				if expiry, parseErr := time.Parse(time.RFC3339Nano, job.ClaimExpiresAt); parseErr == nil && now.After(expiry) {
					eligible = true
				}
			}
			if !eligible {
				continue
			}
			job.Status = "claimed"
			job.ClaimedBy = hostID
			job.ClaimedAt = now.Format(time.RFC3339Nano)
			job.ClaimExpiresAt = now.Add(10 * time.Minute).Format(time.RFC3339Nano)
			job.UpdatedAt = job.ClaimedAt
			job.Result = nil
			job.Error = ""
			if err := writeHostBridgeJSON(filepath.Join(root, "jobs", job.ID+".json"), job); err != nil {
				return err
			}
			copy := job
			claimed = &copy
			break
		}
		return nil
	})
	return claimed, err
}

func CompleteHostBridgeJob(hostID, jobID string, result HostJobResult, jobErr string) (HostJob, error) {
	hostID, err := validateHostBridgeID(hostID)
	if err != nil {
		return HostJob{}, err
	}
	jobID, err = validateHostBridgeID(jobID)
	if err != nil {
		return HostJob{}, err
	}
	result.Output = strings.TrimSpace(result.Output)
	jobErr = strings.TrimSpace(jobErr)
	if len(result.Output) > 16<<10 {
		result.Output = result.Output[:16<<10] + "\n… host output truncated by Workbench …"
	}
	if len(jobErr) > 4096 {
		jobErr = jobErr[:4096] + "…"
	}
	if LooksSecret(result.Output) || LooksSecret(jobErr) {
		return HostJob{}, errors.New("host job completion was withheld because it resembled secret material")
	}
	var job HostJob
	err = withHostBridgeLock(func(root string) error {
		path := filepath.Join(root, "jobs", jobID+".json")
		if err := readHostBridgeJSON(path, &job); err != nil {
			return err
		}
		if job.Status != "claimed" || job.ClaimedBy != hostID || job.HostID != hostID {
			return errors.New("host job is not claimed by this host")
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		job.UpdatedAt = now
		job.ClaimExpiresAt = ""
		if jobErr != "" || result.ExitCode != 0 {
			job.Status = "failed"
			job.Error = jobErr
			if job.Error == "" {
				job.Error = fmt.Sprintf("host tool exited with code %d", result.ExitCode)
			}
		} else {
			job.Status = "completed"
		}
		job.Result = &result
		return writeHostBridgeJSON(path, job)
	})
	if errors.Is(err, os.ErrNotExist) {
		return HostJob{}, errors.New("host job not found")
	}
	return job, err
}
