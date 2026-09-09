package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOverrideRinCanonicalExportUsesDedicatedSubmission(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_HOST_BRIDGE_STATE_DIR", root)
	host, err := RecordHostBridgeHeartbeat(HostBridgeHeartbeat{
		HostID: "windows_test_rin_export", Label: "test Windows host",
		Platform: HostBridgePlatformWindows, Arch: "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SubmitHostBridgeJob(host.HostID, HostJobSpec{Tool: HostBridgeToolUnreal, Operation: HostBridgeOperationOverrideRinCanonicalExport}); err == nil {
		t.Fatal("generic host submission unexpectedly accepted the project-specific Rin export")
	}
	job, err := SubmitOverrideRinCanonicalExportJob(host.HostID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Spec.Tool != HostBridgeToolUnreal || job.Spec.Operation != HostBridgeOperationOverrideRinCanonicalExport || job.Status != "queued" {
		t.Fatalf("unexpected sealed job: %+v", job)
	}
}

func TestValidateOverrideRinExportManifestRequiresExactIdentityAndHashes(t *testing.T) {
	fbx := filepath.Join(t.TempDir(), overrideRinFBXName)
	payload := make([]byte, 4096)
	copy(payload, []byte("Kaydara FBX Binary  \x00\x1a\x00"))
	if err := os.WriteFile(fbx, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	fbxSHA := hex.EncodeToString(sum[:])
	scriptSHA := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	manifest := overrideRinExportManifest{
		SchemaVersion: 1, SourceSHA: overrideRinSourceSHA, CreatedUTC: "2026-09-09T10:00:00+00:00", ScriptSHA256: scriptSHA,
		EngineVersion: "5.8.1", Body: overrideRinBody, Face: overrideRinFace,
		Groom: overrideRinGroom, Blueprint: overrideRinBlueprint,
		InputExportCompleted: true, VisualAcceptance: "not_assessed", AnimationAcceptance: "not_assessed",
		Skeleton: "/Game/MetaHumans/Common/Common/Skeletons/metahuman_base_skel.metahuman_base_skel",
		Materials: []map[string]any{{"slot": "body", "material": "/Game/Test"}},
		Assets: []map[string]any{{"path": "Content/MetaHumans/MHC_RinVale/Body/Test.uasset", "sha256": fbxSHA, "bytes": 4096}},
		FBX: overrideRinFBXManifest{Path: overrideRinFBXName, Bytes: int64(len(payload)), SHA256: fbxSHA, Options: map[string]any{"ascii": false}},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	got, err := validateOverrideRinExportManifest(raw, fbx, scriptSHA)
	if err != nil {
		t.Fatal(err)
	}
	if !got.InputExportCompleted || got.VisualAcceptance != "not_assessed" {
		t.Fatalf("wrong acceptance result: %+v", got)
	}

	manifest.Face = "/Game/Proxy/Face"
	raw, _ = json.Marshal(manifest)
	if _, err := validateOverrideRinExportManifest(raw, fbx, scriptSHA); err == nil {
		t.Fatal("proxy face was accepted")
	}
	manifest.Face = overrideRinFace
	manifest.VisualAcceptance = "accepted"
	raw, _ = json.Marshal(manifest)
	if _, err := validateOverrideRinExportManifest(raw, fbx, scriptSHA); err == nil {
		t.Fatal("automatic visual acceptance was accepted")
	}
}

func TestValidateOverrideRinExportManifestRejectsAlteredFBX(t *testing.T) {
	fbx := filepath.Join(t.TempDir(), overrideRinFBXName)
	if err := os.WriteFile(fbx, make([]byte, 2048), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := overrideRinExportManifest{
		SchemaVersion: 1, SourceSHA: overrideRinSourceSHA, CreatedUTC: "2026-09-09T10:00:00+00:00",
		ScriptSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Body: overrideRinBody, Face: overrideRinFace, Groom: overrideRinGroom, Blueprint: overrideRinBlueprint,
		InputExportCompleted: true, VisualAcceptance: "not_assessed", AnimationAcceptance: "not_assessed",
		Skeleton: "/Game/Skeleton", Assets: []map[string]any{{"path": "x"}},
		FBX: overrideRinFBXManifest{Path: overrideRinFBXName, Bytes: 2048, SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
	}
	raw, _ := json.Marshal(manifest)
	if _, err := validateOverrideRinExportManifest(raw, fbx, manifest.ScriptSHA256); err == nil {
		t.Fatal("altered FBX evidence was accepted")
	}
}
