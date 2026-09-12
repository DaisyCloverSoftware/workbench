//go:build windows

package core

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	overrideRinFieldOutfitReviewJPEGMaxBytes = int64(7500)
	overrideRinFieldOutfitReviewJSONMaxBytes = 15 << 10
	overrideRinFieldOutfitReviewMetaMaxBytes = 4 << 10
)

type overrideRinFieldOutfitReviewRendererMeta struct {
	SchemaVersion            int      `json:"schema_version"`
	Width                    int      `json:"width"`
	Height                   int      `json:"height"`
	JPEGQuality              int      `json:"jpeg_quality"`
	Views                    []string `json:"views"`
	SelectedObjectCount      int      `json:"selected_object_count"`
	ExternalImagesSuppressed int      `json:"external_images_suppressed"`
}

func runOverrideRinFieldOutfitReviewCapture(ctx context.Context, jobID string) (string, error) {
	jobID, err := validateHostBridgeID(jobID)
	if err != nil || !strings.HasPrefix(strings.ToLower(jobID), "hostjob_") {
		return "", errors.New("Rin field-outfit review job id is invalid")
	}
	gitExecutable := findOverrideRinGitExecutable()
	if gitExecutable == "" {
		return "", errors.New("Git for Windows is not installed in an allowlisted location")
	}
	blenderExecutable := findBlenderExecutable()
	if blenderExecutable == "" {
		return "", errors.New("Blender is not installed in an allowlisted Windows location")
	}
	root, err := validateOverrideRinWardrobeReposRoot(overrideRinWardrobeReposRoot)
	if err != nil {
		return "", err
	}
	worktree, head, err := findOverrideRinHardlineWorktree(ctx, gitExecutable, root)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(head, overrideRinFieldOutfitReviewExpectedHead) {
		return "", errors.New("the protected Hardline worktree HEAD moved from the sealed review-capture source")
	}

	before, err := overrideRinFieldOutfitStatus(ctx, gitExecutable, worktree)
	if err != nil {
		return "", err
	}
	tracked, err := overrideRinFieldOutfitTrackedSet(ctx, gitExecutable, worktree)
	if err != nil {
		return "", err
	}
	stagedFiles, stageRoot, stageCreated, err := overrideRinFieldOutfitReviewStagePinnedFiles(ctx, worktree, tracked)
	if err != nil {
		return "", err
	}
	capture, err := overrideRinFieldOutfitRenderReview(ctx, blenderExecutable, stageRoot, jobID)
	if err != nil {
		return "", err
	}
	if err := overrideRinFieldOutfitReviewReverifyPinned(ctx, worktree, tracked); err != nil {
		return "", err
	}
	headAfter, err := outputOverrideRinWardrobeGit(ctx, gitExecutable, worktree, "rev-parse", "HEAD")
	if err != nil || !strings.EqualFold(strings.TrimSpace(headAfter), overrideRinFieldOutfitReviewExpectedHead) {
		return "", errors.New("the protected Hardline worktree HEAD changed during review capture")
	}
	after, err := overrideRinFieldOutfitStatus(ctx, gitExecutable, worktree)
	if err != nil {
		return "", err
	}
	if before.SHA256 != after.SHA256 || before.Records != after.Records {
		return "", errors.New("the protected Hardline worktree changed during review capture")
	}

	result := overrideRinFieldOutfitReviewCaptureResult{
		SchemaVersion:                 1,
		CandidateID:                   overrideRinFieldOutfitReviewStageID,
		ManifestSHA256:                overrideRinFieldOutfitReviewManifestSHA256,
		Branch:                        overrideRinWardrobeBranch,
		HeadSHA:                       overrideRinFieldOutfitReviewExpectedHead,
		ReadOnlySource:                true,
		MutationPerformedSource:       false,
		ProtectedStatusSHA256Before:   before.SHA256,
		ProtectedStatusSHA256After:    after.SHA256,
		ProtectedStatusRecordsBefore:  before.Records,
		ProtectedStatusRecordsAfter:   after.Records,
		ProtectedWorktreeUnchanged:    true,
		StageCreated:                  stageCreated,
		StagedFiles:                   stagedFiles,
		ReviewCapture:                 capture,
		VisualAcceptance:              "not_assessed",
		AnimationAcceptance:           "not_assessed",
		AAAAcceptance:                 false,
	}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > overrideRinFieldOutfitReviewJSONMaxBytes {
		return "", errors.New("Rin field-outfit review capture exceeded its bounded host result")
	}
	lower := strings.ToLower(string(encoded))
	if strings.Contains(lower, strings.ToLower(filepath.Clean(worktree))) || strings.Contains(lower, strings.ToLower(filepath.Clean(stageRoot))) {
		return "", errors.New("Rin field-outfit review result contained a local absolute path")
	}
	return string(encoded), nil
}

func overrideRinFieldOutfitReviewStagePinnedFiles(ctx context.Context, worktree string, tracked map[string]bool) ([]overrideRinFieldOutfitStagedFile, string, bool, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return nil, "", false, errors.New("Workbench recovery cache is unavailable")
	}
	parent := filepath.Join(cacheRoot, "Workbench", "recovery", "rin-field-outfit-review", "v1")
	stageRoot := filepath.Join(parent, overrideRinFieldOutfitReviewStageID)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return nil, "", false, errors.New("Workbench could not prepare the field-outfit review cache")
	}
	if info, statErr := os.Lstat(stageRoot); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, "", false, errors.New("existing field-outfit review stage is not a regular directory")
		}
		files, err := overrideRinFieldOutfitReviewVerifyStage(ctx, worktree, stageRoot, tracked)
		return files, stageRoot, false, err
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, "", false, errors.New("field-outfit review stage could not be inspected")
	}

	temp, err := os.MkdirTemp(parent, ".creating-field-outfit-review-")
	if err != nil {
		return nil, "", false, errors.New("Workbench could not create a field-outfit review staging directory")
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(temp)
		}
	}()
	for _, pin := range overrideRinFieldOutfitReviewPinnedFiles {
		if err := ctx.Err(); err != nil {
			return nil, "", false, errors.New("field-outfit review staging was cancelled")
		}
		source, err := overrideRinFieldOutfitPinnedSourcePath(worktree, pin)
		if err != nil {
			return nil, "", false, err
		}
		if tracked[strings.ToLower(pin.Path)] {
			return nil, "", false, errors.New("a sealed review-capture source unexpectedly became tracked")
		}
		if err := overrideRinFieldOutfitVerifyFile(source, pin); err != nil {
			return nil, "", false, err
		}
		relative := strings.TrimPrefix(pin.Path, overrideRinFieldOutfitSubtree+"/")
		destination := filepath.Join(temp, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return nil, "", false, errors.New("Workbench could not prepare a review-capture staged path")
		}
		if err := overrideRinFieldOutfitCopyExact(source, destination, pin); err != nil {
			return nil, "", false, err
		}
		if err := overrideRinFieldOutfitVerifyFile(source, pin); err != nil {
			return nil, "", false, errors.New("a protected review-capture source changed while being copied")
		}
	}
	if err := os.Rename(temp, stageRoot); err != nil {
		if _, statErr := os.Stat(stageRoot); statErr != nil {
			return nil, "", false, errors.New("Workbench could not publish the field-outfit review stage")
		}
		files, verifyErr := overrideRinFieldOutfitReviewVerifyStage(ctx, worktree, stageRoot, tracked)
		return files, stageRoot, false, verifyErr
	}
	keep = true
	files, err := overrideRinFieldOutfitReviewVerifyStage(ctx, worktree, stageRoot, tracked)
	return files, stageRoot, true, err
}

func overrideRinFieldOutfitReviewVerifyStage(ctx context.Context, worktree, stageRoot string, tracked map[string]bool) ([]overrideRinFieldOutfitStagedFile, error) {
	files := make([]overrideRinFieldOutfitStagedFile, 0, len(overrideRinFieldOutfitReviewPinnedFiles))
	for _, pin := range overrideRinFieldOutfitReviewPinnedFiles {
		if err := ctx.Err(); err != nil {
			return nil, errors.New("field-outfit review stage verification was cancelled")
		}
		source, err := overrideRinFieldOutfitPinnedSourcePath(worktree, pin)
		if err != nil {
			return nil, err
		}
		if tracked[strings.ToLower(pin.Path)] {
			return nil, errors.New("a sealed review-capture source unexpectedly became tracked")
		}
		if err := overrideRinFieldOutfitVerifyFile(source, pin); err != nil {
			return nil, err
		}
		relative := strings.TrimPrefix(pin.Path, overrideRinFieldOutfitSubtree+"/")
		destination := filepath.Join(stageRoot, filepath.FromSlash(relative))
		if err := overrideRinFieldOutfitVerifyFile(destination, pin); err != nil {
			return nil, errors.New("existing field-outfit review stage does not match the sealed source bytes")
		}
		files = append(files, overrideRinFieldOutfitStagedFile{
			Role: pin.Role, Path: pin.Path, Bytes: pin.Bytes, SHA256: pin.SHA256, Tracked: false, Staged: true,
		})
	}
	return files, nil
}

func overrideRinFieldOutfitReviewReverifyPinned(ctx context.Context, worktree string, tracked map[string]bool) error {
	for _, pin := range overrideRinFieldOutfitReviewPinnedFiles {
		if err := ctx.Err(); err != nil {
			return errors.New("field-outfit review source reverification was cancelled")
		}
		if tracked[strings.ToLower(pin.Path)] {
			return errors.New("a sealed review-capture source unexpectedly became tracked")
		}
		source, err := overrideRinFieldOutfitPinnedSourcePath(worktree, pin)
		if err != nil {
			return err
		}
		if err := overrideRinFieldOutfitVerifyFile(source, pin); err != nil {
			return err
		}
	}
	return nil
}

func overrideRinFieldOutfitRenderReview(ctx context.Context, blenderExecutable, stageRoot, jobID string) (overrideRinFieldOutfitReviewCapture, error) {
	pin, ok := overrideRinFieldOutfitReviewPinnedByRole("integrated_outfit")
	if !ok {
		return overrideRinFieldOutfitReviewCapture{}, errors.New("review-capture integrated outfit pin is unavailable")
	}
	relative := strings.TrimPrefix(pin.Path, overrideRinFieldOutfitSubtree+"/")
	blendPath := filepath.Join(stageRoot, filepath.FromSlash(relative))
	if err := overrideRinFieldOutfitVerifyFile(blendPath, pin); err != nil {
		return overrideRinFieldOutfitReviewCapture{}, errors.New("review-capture staged blend failed verification")
	}

	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return overrideRinFieldOutfitReviewCapture{}, errors.New("Workbench recovery cache is unavailable")
	}
	outputRoot := filepath.Join(cacheRoot, "Workbench", "recovery", "rin-field-outfit-review", "output", jobID)
	if err := os.MkdirAll(outputRoot, 0o700); err != nil {
		return overrideRinFieldOutfitReviewCapture{}, errors.New("Workbench could not prepare a review-capture output directory")
	}
	defer os.RemoveAll(outputRoot)
	scriptPath := filepath.Join(outputRoot, "render_review.py")
	imagePath := filepath.Join(outputRoot, "rin_field_outfit_review.jpg")
	metaPath := filepath.Join(outputRoot, "review_meta.json")
	if err := os.WriteFile(scriptPath, []byte(overrideRinFieldOutfitReviewRenderer), 0o600); err != nil {
		return overrideRinFieldOutfitReviewCapture{}, errors.New("Workbench could not prepare the fixed review renderer")
	}
	name, args, err := overrideRinFieldOutfitReviewInvocation(blenderExecutable, blendPath, scriptPath, imagePath, metaPath, stageRoot)
	if err != nil {
		return overrideRinFieldOutfitReviewCapture{}, err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	configureChildProcess(cmd, false)
	stdout := &overrideRinFieldOutfitBoundedBuffer{max: 32 << 10}
	stderr := &overrideRinFieldOutfitBoundedBuffer{max: 32 << 10}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	runErr := cmd.Run()
	if runErr != nil || stdout.overflow || stderr.overflow {
		if ctx.Err() != nil {
			return overrideRinFieldOutfitReviewCapture{}, errors.New("Rin field-outfit review render timed out")
		}
		return overrideRinFieldOutfitReviewCapture{}, errors.New("Rin field-outfit review render failed")
	}
	return overrideRinFieldOutfitReadReviewCapture(imagePath, metaPath)
}

func overrideRinFieldOutfitReviewInvocation(executable, blendPath, scriptPath, imagePath, metaPath, stageRoot string) (string, []string, error) {
	name, _, err := blenderVersionInvocation(executable)
	if err != nil {
		return "", nil, err
	}
	for _, path := range []string{blendPath, scriptPath, imagePath, metaPath, stageRoot} {
		if strings.TrimSpace(path) == "" || strings.ContainsAny(path, "\r\n\x00") || !filepath.IsAbs(path) {
			return "", nil, errors.New("Rin field-outfit review invocation contains an invalid internal path")
		}
	}
	if !strings.HasPrefix(strings.ToLower(filepath.Clean(blendPath)), strings.ToLower(filepath.Clean(stageRoot))+string(os.PathSeparator)) {
		return "", nil, errors.New("Rin field-outfit review blend is outside the isolated stage")
	}
	return name, []string{
		"--background",
		"--factory-startup",
		"--disable-autoexec",
		blendPath,
		"--python", scriptPath,
		"--",
		imagePath,
		metaPath,
		filepath.Clean(stageRoot),
	}, nil
}

func overrideRinFieldOutfitReadReviewCapture(imagePath, metaPath string) (overrideRinFieldOutfitReviewCapture, error) {
	info, err := os.Lstat(imagePath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > overrideRinFieldOutfitReviewJPEGMaxBytes {
		return overrideRinFieldOutfitReviewCapture{}, errors.New("Rin field-outfit review image is outside its bounded file contract")
	}
	data, err := os.ReadFile(imagePath)
	if err != nil || int64(len(data)) != info.Size() || len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 || data[len(data)-2] != 0xff || data[len(data)-1] != 0xd9 {
		return overrideRinFieldOutfitReviewCapture{}, errors.New("Rin field-outfit review image is not a stable JPEG")
	}
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil || len(metaBytes) == 0 || len(metaBytes) > overrideRinFieldOutfitReviewMetaMaxBytes {
		return overrideRinFieldOutfitReviewCapture{}, errors.New("Rin field-outfit review metadata is unavailable or oversized")
	}
	var meta overrideRinFieldOutfitReviewRendererMeta
	dec := json.NewDecoder(strings.NewReader(string(metaBytes)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&meta); err != nil || meta.SchemaVersion != 1 || meta.Width != 480 || meta.Height != 320 || len(meta.Views) != 3 {
		return overrideRinFieldOutfitReviewCapture{}, errors.New("Rin field-outfit review metadata failed validation")
	}
	if strings.Join(meta.Views, ",") != "front,three_quarter,back" || meta.JPEGQuality < 10 || meta.JPEGQuality > 60 || meta.SelectedObjectCount < 1 || meta.ExternalImagesSuppressed < 0 {
		return overrideRinFieldOutfitReviewCapture{}, errors.New("Rin field-outfit review metadata is outside its sealed bounds")
	}
	sum := sha256.Sum256(data)
	return overrideRinFieldOutfitReviewCapture{
		MIME:                     "image/jpeg",
		Width:                    meta.Width,
		Height:                   meta.Height,
		Bytes:                    int64(len(data)),
		SHA256:                   hex.EncodeToString(sum[:]),
		Views:                    append([]string(nil), meta.Views...),
		JPEGQuality:              meta.JPEGQuality,
		SelectedObjectCount:      meta.SelectedObjectCount,
		ExternalImagesSuppressed: meta.ExternalImagesSuppressed,
		ImageBase64:              base64.StdEncoding.EncodeToString(data),
	}, nil
}

const overrideRinFieldOutfitReviewRenderer = `import bpy, json, math, os, sys
from mathutils import Vector

argv = sys.argv
if '--' not in argv or len(argv[argv.index('--')+1:]) != 3:
    raise RuntimeError('sealed review renderer requires internal output, metadata and stage paths')
out_path, meta_path, stage_root = argv[argv.index('--')+1:]
out_path = os.path.abspath(out_path)
meta_path = os.path.abspath(meta_path)
stage_root = os.path.abspath(stage_root)

if bpy.data.libraries:
    raise RuntimeError('sealed review renderer does not allow linked external blend libraries')

helper_tokens = ('ground','floor','grid','plane','reference','proxy','collision','ucx_','socket','locator')
meshes = [o for o in bpy.context.scene.objects if o.type in {'MESH','CURVE','SURFACE'} and not o.hide_render]
selected = [o for o in meshes if not any(t in o.name.lower() for t in helper_tokens)]
if not selected:
    selected = meshes
if not selected:
    raise RuntimeError('sealed review renderer found no renderable outfit objects')
selected_names = {o.name for o in selected}
for o in meshes:
    if o.name not in selected_names:
        o.hide_render = True

external_suppressed = 0
for image in bpy.data.images:
    if image.packed_file is not None or image.source in {'GENERATED','VIEWER'} or not image.filepath:
        continue
    resolved = os.path.abspath(bpy.path.abspath(image.filepath))
    try:
        inside = os.path.commonpath((stage_root, resolved)) == stage_root
    except Exception:
        inside = False
    if not inside:
        external_suppressed += 1
        image.filepath = ''
        image.source = 'GENERATED'
        image.generated_width = 8
        image.generated_height = 8
        image.generated_color = (0.45, 0.45, 0.45, 1.0)

points = []
for obj in selected:
    for corner in obj.bound_box:
        points.append(obj.matrix_world @ Vector(corner))
mins = Vector((min(p.x for p in points), min(p.y for p in points), min(p.z for p in points)))
maxs = Vector((max(p.x for p in points), max(p.y for p in points), max(p.z for p in points)))
center = (mins + maxs) * 0.5
size = maxs - mins
height = max(size.z, 0.01)
width = max(size.x, size.y, 0.01)

for obj in list(bpy.context.scene.objects):
    if obj.type in {'LIGHT','CAMERA'}:
        obj.hide_render = True

world = bpy.context.scene.world or bpy.data.worlds.new('WorkbenchReviewWorld')
bpy.context.scene.world = world
world.use_nodes = True
bg = world.node_tree.nodes.get('Background')
if bg:
    bg.inputs['Color'].default_value = (0.055, 0.055, 0.055, 1.0)
    bg.inputs['Strength'].default_value = 0.35

def add_area(name, location, energy, size):
    data = bpy.data.lights.new(name=name, type='AREA')
    data.energy = energy
    data.shape = 'DISK'
    data.size = size
    obj = bpy.data.objects.new(name, data)
    bpy.context.collection.objects.link(obj)
    obj.location = location
    direction = center - obj.location
    obj.rotation_euler = direction.to_track_quat('-Z','Y').to_euler()
    return obj

radius = max(height, width)
add_area('WorkbenchReviewKey', center + Vector((-radius*0.9, -radius*1.2, radius*0.8)), 900.0, radius*0.8)
add_area('WorkbenchReviewFill', center + Vector((radius*0.9, -radius*0.7, radius*0.35)), 450.0, radius*0.7)
add_area('WorkbenchReviewRim', center + Vector((0, radius*1.0, radius*0.8)), 700.0, radius*0.6)

cam_data = bpy.data.cameras.new('WorkbenchReviewCamera')
cam = bpy.data.objects.new('WorkbenchReviewCamera', cam_data)
bpy.context.collection.objects.link(cam)
bpy.context.scene.camera = cam
cam_data.type = 'ORTHO'
cam_data.ortho_scale = max(height * 1.12, width * 2.15)
distance = max(radius * 3.0, 1.0)

scene = bpy.context.scene
scene.render.engine = 'BLENDER_EEVEE_NEXT'
scene.render.resolution_x = 160
scene.render.resolution_y = 320
scene.render.resolution_percentage = 100
scene.render.film_transparent = False
scene.render.image_settings.file_format = 'PNG'
scene.render.use_file_extension = True

views = [
    ('front', Vector((0, -distance, 0))),
    ('three_quarter', Vector((distance*0.72, -distance*0.72, 0))),
    ('back', Vector((0, distance, 0))),
]
paths = []
for label, offset in views:
    cam.location = center + offset
    cam.rotation_euler = (center - cam.location).to_track_quat('-Z','Y').to_euler()
    path = os.path.join(os.path.dirname(out_path), 'view_' + label + '.png')
    scene.render.filepath = path
    bpy.ops.render.render(write_still=True)
    paths.append(path)

w, h = 160, 320
sheet_w = w * 3
sheet = bpy.data.images.new('WorkbenchReviewSheet', width=sheet_w, height=h, alpha=False)
dst = [0.0] * (sheet_w * h * 4)
for index, path in enumerate(paths):
    img = bpy.data.images.load(path, check_existing=False)
    src = list(img.pixels[:])
    if img.size[0] != w or img.size[1] != h:
        raise RuntimeError('sealed review renderer produced an unexpected view size')
    for y in range(h):
        src_start = y * w * 4
        src_end = src_start + w * 4
        dst_start = (y * sheet_w + index * w) * 4
        dst[dst_start:dst_start + w * 4] = src[src_start:src_end]
sheet.pixels[:] = dst

scene.render.image_settings.file_format = 'JPEG'
chosen = None
for quality in (55, 45, 35, 25, 18, 12, 10):
    scene.render.image_settings.quality = quality
    sheet.save_render(out_path, scene=scene)
    n = os.path.getsize(out_path)
    if 700 <= n <= 7500:
        chosen = quality
        break
if chosen is None:
    raise RuntimeError('sealed review renderer could not fit the bounded JPEG result')

for path in paths:
    try:
        os.remove(path)
    except Exception:
        pass

meta = {
    'schema_version': 1,
    'width': sheet_w,
    'height': h,
    'jpeg_quality': chosen,
    'views': [v[0] for v in views],
    'selected_object_count': len(selected),
    'external_images_suppressed': external_suppressed,
}
with open(meta_path, 'w', encoding='utf-8', newline='\n') as f:
    json.dump(meta, f, separators=(',',':'), sort_keys=True)
`
