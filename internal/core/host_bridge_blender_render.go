package core

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	HostBridgeOperationBlenderSmokeRender = "render_smoke"
	maxBlenderSmokeRenderBytes            = int64(32 << 20)
)

var blenderSmokePNGSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

func SubmitBlenderSmokeRenderJob(hostID string) (HostJob, error) {
	hostID, err := validateHostBridgeID(hostID)
	if err != nil { return HostJob{}, err }
	jobID, err := newHostJobID()
	if err != nil { return HostJob{}, err }
	now := time.Now().UTC().Format(time.RFC3339Nano)
	job := HostJob{ID: jobID, HostID: hostID, Spec: HostJobSpec{Tool: HostBridgeToolBlender, Operation: HostBridgeOperationBlenderSmokeRender}, Status: "queued", CreatedAt: now, UpdatedAt: now}
	err = withHostBridgeLock(func(root string) error {
		var host HostBridgeHost
		if err := readHostBridgeJSON(filepath.Join(root, "hosts", hostID+".json"), &host); err != nil {
			if errors.Is(err, os.ErrNotExist) { return errors.New("target host is not registered") }
			return err
		}
		return writeHostBridgeJSON(filepath.Join(root, "jobs", job.ID+".json"), job)
	})
	return job, err
}

func blenderSmokeRenderInvocation(executable, outputPrefix string) (string, []string, error) {
	name, _, err := blenderVersionInvocation(executable)
	if err != nil { return "", nil, err }
	outputPrefix = strings.TrimSpace(outputPrefix)
	if outputPrefix == "" || strings.ContainsAny(outputPrefix, "\r\n\x00") { return "", nil, errors.New("Blender smoke render output path is invalid") }
	if !filepath.IsAbs(outputPrefix) || filepath.Base(outputPrefix) != "smoke_" { return "", nil, errors.New("Blender smoke render output path is outside the fixed Workbench shape") }
	// Factory startup keeps host rendering deterministic. Configure Cycles and
	// NVIDIA GPU devices explicitly in-process so headless renders never depend
	// on GUI preferences saved by a particular Blender installation/user.
	gpuSetup := "import bpy; s=bpy.context.scene; s.render.engine='BLENDER_EEVEE_NEXT'; " +
		"prefs=bpy.context.preferences.addons.get('cycles'); " +
		"exec(\"prefs.preferences.compute_device_type='OPTIX'\\ntry: prefs.preferences.get_devices()\\nexcept: pass\\nfor d in prefs.preferences.devices: d.use=(d.type!='CPU')\\ns.render.engine='CYCLES'\\ns.cycles.device='GPU'\") if prefs else None"
	return name, []string{
		"--background",
		"--factory-startup",
		"--disable-autoexec",
		"--python-expr", gpuSetup,
		"--render-output", outputPrefix,
		"--render-format", "PNG",
		"--render-frame", "1",
	}, nil
}

func blenderSmokeRenderPaths(jobID string) (dir, outputPrefix, expectedFile string, err error) {
	jobID, err = validateHostBridgeID(jobID); if err != nil { return "", "", "", err }
	cache, err := os.UserCacheDir(); if err != nil { return "", "", "", err }
	dir = filepath.Join(cache, "Workbench", "host-bridge", "renders", jobID)
	outputPrefix = filepath.Join(dir, "smoke_")
	expectedFile = filepath.Join(dir, "smoke_0001.png")
	return dir, outputPrefix, expectedFile, nil
}

func verifyBlenderSmokeRender(path string) (string, error) {
	info, err := os.Lstat(path); if err != nil { return "", errors.New("Blender smoke render did not produce the expected PNG") }
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() { return "", errors.New("Blender smoke render output is not a regular file") }
	if info.Size() <= 0 || info.Size() > maxBlenderSmokeRenderBytes { return "", errors.New("Blender smoke render output size is outside the allowed range") }
	f, err := os.Open(path); if err != nil { return "", errors.New("Blender smoke render output could not be verified") }; defer f.Close()
	signature := make([]byte, len(blenderSmokePNGSignature))
	if _, err := io.ReadFull(f, signature); err != nil || string(signature) != string(blenderSmokePNGSignature) { return "", errors.New("Blender smoke render output is not a PNG") }
	if _, err := f.Seek(0, io.SeekStart); err != nil { return "", errors.New("Blender smoke render output could not be verified") }
	h := sha256.New(); n, err := io.Copy(h, io.LimitReader(f, maxBlenderSmokeRenderBytes+1))
	if err != nil || n != info.Size() || n > maxBlenderSmokeRenderBytes { return "", errors.New("Blender smoke render output changed while being verified") }
	return fmt.Sprintf("Blender smoke render complete: artifact=smoke_0001.png bytes=%d sha256=%x", n, h.Sum(nil)), nil
}
