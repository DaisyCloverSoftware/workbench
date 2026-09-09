package core

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
)

const (
	HostBridgeOperationOverrideRinWardrobeInventory = "override_rin_wardrobe_inventory_v1"
	overrideRinWardrobeReposRoot                     = `F:\Repos`
	overrideRinWardrobeBranch                        = "agent/hardline-premium-slice-20260822"
)

const overrideRinWardrobeInventoryTimeout = 3 * time.Minute

var overrideRinWardrobeExtensions = map[string]bool{
	".blend": true, ".fbx": true, ".obj": true, ".gltf": true, ".glb": true,
	".uasset": true, ".uexp": true, ".ubulk": true,
	".png": true, ".tga": true, ".exr": true, ".jpg": true, ".jpeg": true, ".psd": true,
}

type overrideRinWardrobeCandidate struct {
	Path       string `json:"path"`
	Extension  string `json:"extension"`
	Bytes      int64  `json:"bytes"`
	SHA256     string `json:"sha256,omitempty"`
	HashStatus string `json:"hash_status"`
	Tracked    bool   `json:"tracked"`
	Score      int    `json:"score"`
}

type overrideRinWardrobeInventoryResult struct {
	SchemaVersion       int                            `json:"schema_version"`
	ArtifactID          string                         `json:"artifact_id"`
	ReposRoot           string                         `json:"repos_root"`
	WorktreeRoot        string                         `json:"worktree_root"`
	Branch              string                         `json:"branch"`
	HeadSHA             string                         `json:"head_sha"`
	ReadOnly            bool                           `json:"read_only"`
	MutationPerformed   bool                           `json:"mutation_performed"`
	MatchedCount        int                            `json:"matched_count"`
	ReturnedCount       int                            `json:"returned_count"`
	OversizedCount      int                            `json:"oversized_count"`
	CandidateTruncated  bool                           `json:"candidate_truncated"`
	Candidates          []overrideRinWardrobeCandidate `json:"candidates"`
	VisualAcceptance    string                         `json:"visual_acceptance"`
	AnimationAcceptance string                         `json:"animation_acceptance"`
}

// SubmitOverrideRinWardrobeInventoryJob creates one sealed, read-only recovery
// job. The caller selects only an already-registered Windows host. The Windows
// executable fixes the repository root, Override origin, Hardline branch,
// extensions, filename vocabulary, traversal bounds and returned metadata.
func SubmitOverrideRinWardrobeInventoryJob(hostID string) (HostJob, error) {
	hostID, err := validateHostBridgeID(hostID)
	if err != nil {
		return HostJob{}, err
	}
	jobID, err := newHostJobID()
	if err != nil {
		return HostJob{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	job := HostJob{
		ID: jobID, HostID: hostID,
		Spec: HostJobSpec{Tool: HostBridgeToolWorkbench, Operation: HostBridgeOperationOverrideRinWardrobeInventory},
		Status: "queued", CreatedAt: now, UpdatedAt: now,
	}
	err = withHostBridgeLock(func(root string) error {
		var host HostBridgeHost
		if err := readHostBridgeJSON(filepath.Join(root, "hosts", hostID+".json"), &host); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return errors.New("target host is not registered")
			}
			return err
		}
		seen, err := time.Parse(time.RFC3339Nano, host.LastSeen)
		age := time.Since(seen)
		if err != nil || age < 0 || age > 2*time.Minute || !host.Online || host.Platform != HostBridgePlatformWindows {
			return errors.New("a fresh online Windows host heartbeat is required")
		}
		return writeHostBridgeJSON(filepath.Join(root, "jobs", job.ID+".json"), job)
	})
	return job, err
}

func overrideRinWardrobeCandidateScore(path string) int {
	ext := strings.ToLower(filepath.Ext(path))
	if !overrideRinWardrobeExtensions[ext] {
		return 0
	}
	text := strings.ToLower(filepath.ToSlash(path))
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		switch r {
		case '/', '\\', '_', '-', '.', ' ', '(', ')', '[', ']', '{', '}':
			return true
		default:
			return false
		}
	})
	score := 0
	for _, token := range tokens {
		switch token {
		case "rinvale":
			score += 10
		case "rin":
			score += 7
		case "clash":
			score += 10
		case "cargo":
			score += 9
		case "trouser", "trousers", "pant", "pants":
			score += 7
		case "boot", "boots", "combat":
			score += 7
		case "shirt", "tshirt", "tee":
			score += 6
		case "wardrobe", "outfit", "clothing", "garment":
			score += 4
		case "field", "fieldoutfit":
			score += 2
		case "top":
			score += 1
		}
	}
	if score == 0 {
		return 0
	}
	switch ext {
	case ".blend", ".fbx", ".obj", ".gltf", ".glb":
		score += 5
	case ".uasset", ".uexp", ".ubulk":
		score += 3
	default:
		score++
	}
	return score
}
