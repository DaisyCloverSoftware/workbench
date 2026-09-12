package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	HostBridgeOperationOverrideRinFieldOutfitReviewCapture = "override_rin_field_outfit_review_capture_v1"
	overrideRinFieldOutfitReviewExpectedHead               = "3aed03a7d30b987bb76af5fd8230c941a2758e49"
	overrideRinFieldOutfitReviewStageID                    = "rin-field-outfit-review-v1-eec3287f2544d70f"
	overrideRinFieldOutfitReviewManifestSHA256             = "eec3287f2544d70fd6e132f4c2179790e5f9e5cff90ffea6fc74308c14de5927"
)

const overrideRinFieldOutfitReviewTimeout = 8 * time.Minute

var overrideRinFieldOutfitReviewPinnedFiles = []overrideRinFieldOutfitPinnedFile{
	{Role: "integrated_outfit", Path: "SourceAssets/RinWardrobe/AST_RL_CHR_RIN_FIELD_OUTFIT.production.blend", Bytes: 8731474, SHA256: "7ec4d0db94d3f7ad71df357b1e0a801df6a5f873484abddca0803c84c2295469"},
	{Role: "integrated_export", Path: "SourceAssets/RinWardrobe/Exports/SK_RL_Rin_FieldOutfit.fbx", Bytes: 5271308, SHA256: "115515610f608694791ac377f6e7ad8d3e4fbda1b93700443da6c0fbfb7e7f21"},
	{Role: "combat_boots_fit", Path: "SourceAssets/RinWardrobe/FittedBoots/AST_RL_CHR_RIN_BLACK_COMBAT_BOOTS_FIT.blend", Bytes: 2206529, SHA256: "9fa87f121fdddcf2cee530a46e59cb73fb02bc20e15d0505f9516a35898bb9f6"},
	{Role: "cargo_fit", Path: "SourceAssets/RinWardrobe/FittedCargo/AST_RL_CHR_RIN_FUTURISTIC_CARGO_FIT.blend", Bytes: 5753828, SHA256: "fdcf2290e86d70c6f80742de4e86e7441693571cbf885a3d7e2939e99619ccbe"},
	{Role: "cargo_authored_base", Path: "SourceAssets/RinWardrobe/AuthoredBase/AST_RL_CHR_RIN_CARGO_AUTHORED_BASE.blend", Bytes: 3400012, SHA256: "306960e889a5e674777f6077d012dc8bc31e181fdd085665487beb52b4f67d10"},
}

type overrideRinFieldOutfitReviewCapture struct {
	MIME                     string   `json:"mime"`
	Width                    int      `json:"width"`
	Height                   int      `json:"height"`
	Bytes                    int64    `json:"bytes"`
	SHA256                   string   `json:"sha256"`
	Views                    []string `json:"views"`
	JPEGQuality              int      `json:"jpeg_quality"`
	SelectedObjectCount      int      `json:"selected_object_count"`
	ExternalImagesSuppressed int      `json:"external_images_suppressed"`
	ImageBase64              string   `json:"image_base64"`
}

type overrideRinFieldOutfitReviewCaptureResult struct {
	SchemaVersion                 int                                 `json:"schema_version"`
	CandidateID                   string                              `json:"candidate_id"`
	ManifestSHA256                string                              `json:"manifest_sha256"`
	Branch                        string                              `json:"branch"`
	HeadSHA                       string                              `json:"head_sha"`
	ReadOnlySource                bool                                `json:"read_only_source"`
	MutationPerformedSource       bool                                `json:"mutation_performed_source"`
	ProtectedStatusSHA256Before   string                              `json:"protected_status_sha256_before"`
	ProtectedStatusSHA256After    string                              `json:"protected_status_sha256_after"`
	ProtectedStatusRecordsBefore  int                                 `json:"protected_status_records_before"`
	ProtectedStatusRecordsAfter   int                                 `json:"protected_status_records_after"`
	ProtectedWorktreeUnchanged    bool                                `json:"protected_worktree_unchanged"`
	StageCreated                  bool                                `json:"stage_created"`
	StagedFiles                   []overrideRinFieldOutfitStagedFile  `json:"staged_files"`
	ReviewCapture                 overrideRinFieldOutfitReviewCapture `json:"review_capture"`
	VisualAcceptance              string                              `json:"visual_acceptance"`
	AnimationAcceptance           string                              `json:"animation_acceptance"`
	AAAAcceptance                 bool                                `json:"aaa_acceptance"`
}

func SubmitOverrideRinFieldOutfitReviewCaptureJob(hostID string) (HostJob, error) {
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
		Spec: HostJobSpec{Tool: HostBridgeToolWorkbench, Operation: HostBridgeOperationOverrideRinFieldOutfitReviewCapture},
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

func overrideRinFieldOutfitReviewPinnedByRole(role string) (overrideRinFieldOutfitPinnedFile, bool) {
	role = strings.TrimSpace(role)
	for _, pin := range overrideRinFieldOutfitReviewPinnedFiles {
		if pin.Role == role {
			return pin, true
		}
	}
	return overrideRinFieldOutfitPinnedFile{}, false
}
