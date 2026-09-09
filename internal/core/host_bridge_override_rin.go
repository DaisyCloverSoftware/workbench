package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	HostBridgeOperationOverrideRinCanonicalExport = "override_rin_canonical_export_v1"
	overrideRinRepositoryURL                       = "https://github.com/DaisyCloverSoftware/override.git"
	overrideRinSourceSHA                           = "518ecfe4b4f39205d7e78dbe7698854738d74525"
	overrideRinExporterSHA                         = "530bb414f27f030033e38a65da42db6b3963a879"
	overrideRinExporterPath                        = "tools/recovery/export_canonical_rin_inputs.py"
	overrideRinProjectName                         = "OverrideZeroDay.uproject"
	overrideRinEditorTarget                        = "OverrideZeroDayEditor"
	overrideRinEngineAssociation                   = "{04C4EE27-4B30-6770-F3A2-5AA29FF75A0F}"
	overrideRinBody                                = "/Game/MetaHumans/MHC_RinVale/Body/SKM_MHC_RinVale_BodyMesh"
	overrideRinFace                                = "/Game/MetaHumans/MHC_RinVale/Face/SKM_MHC_RinVale_FaceMesh"
	overrideRinGroom                               = "/Game/MetaHumans/MHC_RinVale/Grooms/Hair_S_UpdoBraids"
	overrideRinBlueprint                           = "/Game/MetaHumans/MHC_RinVale/BP_MHC_RinVale"
	overrideRinFBXName                             = "RinVale_canonical_body_reference.fbx"
	overrideRinManifestName                        = "rin-input-export.json"
)

const overrideRinExportTimeout = 45 * time.Minute

type overrideRinFBXManifest struct {
	Path    string         `json:"path"`
	Bytes   int64          `json:"bytes"`
	SHA256  string         `json:"sha256"`
	Options map[string]any `json:"options"`
}

type overrideRinExportManifest struct {
	SchemaVersion       int                    `json:"schema_version"`
	SourceSHA           string                 `json:"source_sha"`
	ScriptSHA256        string                 `json:"script_sha256"`
	EngineVersion       string                 `json:"engine_version"`
	Body                string                 `json:"body"`
	Face                string                 `json:"face"`
	Groom               string                 `json:"groom"`
	Blueprint           string                 `json:"blueprint"`
	InputExportCompleted bool                  `json:"input_export_completed"`
	VisualAcceptance    string                 `json:"visual_acceptance"`
	AnimationAcceptance string                 `json:"animation_acceptance"`
	Skeleton            string                 `json:"skeleton"`
	Materials           []map[string]any       `json:"materials"`
	Assets              []map[string]any       `json:"assets"`
	FBX                 overrideRinFBXManifest `json:"fbx"`
	Error               string                 `json:"error,omitempty"`
}

type overrideRinExportResult struct {
	SchemaVersion       int    `json:"schema_version"`
	ArtifactID          string `json:"artifact_id"`
	SourceSHA           string `json:"source_sha"`
	EngineVersion       string `json:"engine_version"`
	Body                string `json:"body"`
	Face                string `json:"face"`
	Groom               string `json:"groom"`
	Blueprint           string `json:"blueprint"`
	Skeleton            string `json:"skeleton"`
	IdentityAssetCount  int    `json:"identity_asset_count"`
	MaterialSlotCount   int    `json:"material_slot_count"`
	FBXBytes            int64  `json:"fbx_bytes"`
	FBXSHA256           string `json:"fbx_sha256"`
	ManifestSHA256      string `json:"manifest_sha256"`
	InputExportCompleted bool  `json:"input_export_completed"`
	VisualAcceptance    string `json:"visual_acceptance"`
	AnimationAcceptance string `json:"animation_acceptance"`
}

// SubmitOverrideRinCanonicalExportJob creates one project-specific, sealed job.
// Callers can select only the already-registered Windows host. They cannot
// supply a repository, ref, project, script, executable, path, commandlet or
// argument. The Windows side revalidates the same fixed operation before doing
// any work.
func SubmitOverrideRinCanonicalExportJob(hostID string) (HostJob, error) {
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
		Spec: HostJobSpec{Tool: HostBridgeToolUnreal, Operation: HostBridgeOperationOverrideRinCanonicalExport},
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
		if err != nil || time.Since(seen) < 0 || time.Since(seen) > 2*time.Minute || !host.Online || host.Platform != HostBridgePlatformWindows {
			return errors.New("a fresh online Windows host heartbeat is required")
		}
		return writeHostBridgeJSON(filepath.Join(root, "jobs", job.ID+".json"), job)
	})
	return job, err
}

func validateOverrideRinExportManifest(raw []byte, fbxPath, expectedScriptSHA string) (overrideRinExportManifest, error) {
	if len(raw) == 0 || len(raw) > 2<<20 {
		return overrideRinExportManifest{}, errors.New("Rin export manifest is missing or oversized")
	}
	var manifest overrideRinExportManifest
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return overrideRinExportManifest{}, errors.New("Rin export manifest is malformed")
	}
	if manifest.SchemaVersion != 1 || manifest.SourceSHA != overrideRinSourceSHA || !manifest.InputExportCompleted {
		return overrideRinExportManifest{}, errors.New("Rin export manifest does not prove the exact completed source export")
	}
	if manifest.Body != overrideRinBody || manifest.Face != overrideRinFace || manifest.Groom != overrideRinGroom || manifest.Blueprint != overrideRinBlueprint {
		return overrideRinExportManifest{}, errors.New("Rin export manifest names the wrong canonical identity")
	}
	if manifest.VisualAcceptance != "not_assessed" || manifest.AnimationAcceptance != "not_assessed" || strings.TrimSpace(manifest.Error) != "" {
		return overrideRinExportManifest{}, errors.New("Rin export manifest crossed its acceptance boundary")
	}
	if manifest.ScriptSHA256 != expectedScriptSHA || len(manifest.ScriptSHA256) != 64 || manifest.Skeleton == "" || len(manifest.Assets) == 0 {
		return overrideRinExportManifest{}, errors.New("Rin export manifest is missing required provenance")
	}
	if manifest.FBX.Path != overrideRinFBXName || manifest.FBX.Bytes < 1024 || len(manifest.FBX.SHA256) != 64 {
		return overrideRinExportManifest{}, errors.New("Rin export manifest contains invalid FBX evidence")
	}
	info, err := os.Lstat(fbxPath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() != manifest.FBX.Bytes {
		return overrideRinExportManifest{}, errors.New("Rin export FBX is missing or changed")
	}
	actual, err := sha256RegularFile(fbxPath)
	if err != nil || actual != manifest.FBX.SHA256 {
		return overrideRinExportManifest{}, errors.New("Rin export FBX hash does not match its manifest")
	}
	return manifest, nil
}

func sha256RegularFile(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", errors.New("file is missing or not a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	buf := make([]byte, 1024*1024)
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			_, _ = h.Write(buf[:n])
		}
		if errors.Is(readErr, os.ErrClosed) {
			return "", readErr
		}
		if readErr != nil {
			if readErr.Error() == "EOF" {
				break
			}
			return "", readErr
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
