package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	HostBridgeOperationOverrideRinFieldOutfitInspect = "override_rin_field_outfit_inspect_v1"
	overrideRinFieldOutfitExpectedHead               = "3aed03a7d30b987bb76af5fd8230c941a2758e49"
	overrideRinFieldOutfitSubtree                    = "SourceAssets/RinWardrobe"
	overrideRinFieldOutfitStageID                    = "rin-field-outfit-v1-7ec41284367e3e86"
)

const overrideRinFieldOutfitInspectTimeout = 8 * time.Minute

type overrideRinFieldOutfitPinnedFile struct {
	Role   string
	Path   string
	Bytes  int64
	SHA256 string
}

var overrideRinFieldOutfitPinnedFiles = []overrideRinFieldOutfitPinnedFile{
	{Role: "integrated_outfit", Path: "SourceAssets/RinWardrobe/AST_RL_CHR_RIN_FIELD_OUTFIT.production.blend", Bytes: 8784510, SHA256: "7ec41284367e3e8638ba2fec33eeb64608a166b7473c5c42bf71b34b1e22a1db"},
	{Role: "integrated_export", Path: "SourceAssets/RinWardrobe/Exports/SK_RL_Rin_FieldOutfit.fbx", Bytes: 5273468, SHA256: "1155b15eddf94177f52cff5c5ef98f9566b526255edeef22b3a72eb992c8597f"},
	{Role: "combat_boots_fit", Path: "SourceAssets/RinWardrobe/FittedBoots/AST_RL_CHR_RIN_BLACK_COMBAT_BOOTS_FIT.blend", Bytes: 2205834, SHA256: "9fac75894757028aa7ae5066e20cc491920f2886540b710b640f863cf883a39c"},
	{Role: "cargo_fit", Path: "SourceAssets/RinWardrobe/FittedCargo/AST_RL_CHR_RIN_FUTURISTIC_CARGO_FIT.blend", Bytes: 4162814, SHA256: "fdcfba74944900f930fea2c92137a973de8f7dca4559dad4db76d666e67537a2"},
	{Role: "cargo_authored_base", Path: "SourceAssets/RinWardrobe/AuthoredBase/AST_RL_CHR_RIN_CARGO_AUTHORED_BASE.blend", Bytes: 118696, SHA256: "306937dc8ffe22d238104945493860cad7673e3ba6ba0cc864435dd796149dd6"},
}

type overrideRinFieldOutfitStagedFile struct {
	Role    string `json:"role"`
	Path    string `json:"path"`
	Bytes   int64  `json:"bytes"`
	SHA256  string `json:"sha256"`
	Tracked bool   `json:"tracked"`
	Staged  bool   `json:"staged"`
}

type overrideRinFieldOutfitSubtreeFile struct {
	Path      string `json:"path"`
	Extension string `json:"extension"`
	Bytes     int64  `json:"bytes"`
	Tracked   bool   `json:"tracked"`
}

type overrideRinFieldOutfitBlenderObject struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Parent     string    `json:"parent,omitempty"`
	Dimensions []float64 `json:"dimensions,omitempty"`
	Modifiers  []string  `json:"modifiers,omitempty"`
	Materials  []string  `json:"materials,omitempty"`
	ShapeKeys  []string  `json:"shape_keys,omitempty"`
	Vertices   int       `json:"vertices,omitempty"`
	Polygons   int       `json:"polygons,omitempty"`
	Bones      int       `json:"bones,omitempty"`
}

type overrideRinFieldOutfitBlenderAction struct {
	Name       string  `json:"name"`
	FrameStart float64 `json:"frame_start"`
	FrameEnd   float64 `json:"frame_end"`
}

type overrideRinFieldOutfitBlenderInspection struct {
	SchemaVersion  int                                  `json:"schema_version"`
	BlenderVersion string                               `json:"blender_version"`
	ObjectCount    int                                  `json:"object_count"`
	Objects        []overrideRinFieldOutfitBlenderObject `json:"objects"`
	Materials      []string                             `json:"materials"`
	Images         []string                             `json:"images"`
	Libraries      []string                             `json:"libraries"`
	Actions        []overrideRinFieldOutfitBlenderAction `json:"actions"`
	SemanticFlags  map[string]bool                      `json:"semantic_flags"`
	Truncated      bool                                 `json:"truncated"`
}

type overrideRinFieldOutfitInspectResult struct {
	SchemaVersion                int                                  `json:"schema_version"`
	ArtifactID                   string                               `json:"artifact_id"`
	Branch                       string                               `json:"branch"`
	HeadSHA                      string                               `json:"head_sha"`
	ReadOnlySource               bool                                 `json:"read_only_source"`
	MutationPerformedSource      bool                                 `json:"mutation_performed_source"`
	ProtectedStatusSHA256Before  string                               `json:"protected_status_sha256_before"`
	ProtectedStatusSHA256After   string                               `json:"protected_status_sha256_after"`
	ProtectedStatusRecordsBefore int                                  `json:"protected_status_records_before"`
	ProtectedStatusRecordsAfter  int                                  `json:"protected_status_records_after"`
	ProtectedWorktreeUnchanged   bool                                 `json:"protected_worktree_unchanged"`
	StageID                      string                               `json:"stage_id"`
	StageCreated                 bool                                 `json:"stage_created"`
	StagedFiles                  []overrideRinFieldOutfitStagedFile   `json:"staged_files"`
	SubtreeMatchedCount          int                                  `json:"subtree_matched_count"`
	SubtreeReturnedCount         int                                  `json:"subtree_returned_count"`
	SubtreeTruncated             bool                                 `json:"subtree_truncated"`
	SubtreeFiles                 []overrideRinFieldOutfitSubtreeFile  `json:"subtree_files"`
	Blender                      overrideRinFieldOutfitBlenderInspection `json:"blender"`
	VisualAcceptance             string                               `json:"visual_acceptance"`
	AnimationAcceptance          string                               `json:"animation_acceptance"`
	AAAAcceptance                bool                                 `json:"aaa_acceptance"`
}

func SubmitOverrideRinFieldOutfitInspectJob(hostID string) (HostJob, error) {
	hostID, err := validateHostBridgeID(hostID)
	if err != nil {
		return HostJob{}, err
	}
	jobID, err := newHostJobID()
	if err != nil {
		return HostJob{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	job := HostJob{ID: jobID, HostID: hostID, Spec: HostJobSpec{Tool: HostBridgeToolWorkbench, Operation: HostBridgeOperationOverrideRinFieldOutfitInspect}, Status: "queued", CreatedAt: now, UpdatedAt: now}
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

func overrideRinFieldOutfitPinnedByPath(path string) (overrideRinFieldOutfitPinnedFile, bool) {
	path = filepath.ToSlash(strings.TrimSpace(path))
	for _, pin := range overrideRinFieldOutfitPinnedFiles {
		if path == pin.Path {
			return pin, true
		}
	}
	return overrideRinFieldOutfitPinnedFile{}, false
}
