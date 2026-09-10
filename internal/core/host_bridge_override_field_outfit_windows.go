//go:build windows

package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	overrideRinFieldOutfitStatusMaxBytes       = 4 << 20
	overrideRinFieldOutfitSubtreeMaxEntries   = 10000
	overrideRinFieldOutfitSubtreeMaxReturned  = 512
	overrideRinFieldOutfitBlenderJSONMaxBytes = 128 << 10
)

type overrideRinFieldOutfitStatusSnapshot struct {
	SHA256  string
	Records int
}

type overrideRinFieldOutfitBoundedBuffer struct {
	buffer   bytes.Buffer
	max      int
	overflow bool
}

func (b *overrideRinFieldOutfitBoundedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := b.max - b.buffer.Len()
	if remaining <= 0 {
		b.overflow = b.overflow || original > 0
		return original, nil
	}
	if len(p) > remaining {
		_, _ = b.buffer.Write(p[:remaining])
		b.overflow = true
		return original, nil
	}
	_, _ = b.buffer.Write(p)
	return original, nil
}

func runOverrideRinFieldOutfitInspect(ctx context.Context, jobID string) (string, error) {
	jobID, err := validateHostBridgeID(jobID)
	if err != nil || !strings.HasPrefix(strings.ToLower(jobID), "hostjob_") {
		return "", errors.New("Rin field-outfit inspection job id is invalid")
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
	if !strings.EqualFold(head, overrideRinFieldOutfitExpectedHead) {
		return "", errors.New("the protected Hardline worktree HEAD moved from the sealed field-outfit inspection source")
	}

	before, err := overrideRinFieldOutfitStatus(ctx, gitExecutable, worktree)
	if err != nil {
		return "", err
	}
	tracked, err := overrideRinFieldOutfitTrackedSet(ctx, gitExecutable, worktree)
	if err != nil {
		return "", err
	}
	stagedFiles, stageRoot, stageCreated, err := overrideRinFieldOutfitStagePinnedFiles(ctx, worktree, tracked)
	if err != nil {
		return "", err
	}
	subtreeFiles, subtreeCount, subtreeTruncated, err := overrideRinFieldOutfitInventorySubtree(ctx, worktree, tracked)
	if err != nil {
		return "", err
	}
	blender, err := overrideRinFieldOutfitInspectBlend(ctx, blenderExecutable, stageRoot)
	if err != nil {
		return "", err
	}

	if err := overrideRinFieldOutfitReverifyPinned(ctx, worktree, tracked); err != nil {
		return "", err
	}
	headAfter, err := outputOverrideRinWardrobeGit(ctx, gitExecutable, worktree, "rev-parse", "HEAD")
	if err != nil || !strings.EqualFold(strings.TrimSpace(headAfter), overrideRinFieldOutfitExpectedHead) {
		return "", errors.New("the protected Hardline worktree HEAD changed during field-outfit inspection")
	}
	after, err := overrideRinFieldOutfitStatus(ctx, gitExecutable, worktree)
	if err != nil {
		return "", err
	}
	unchanged := before.SHA256 == after.SHA256 && before.Records == after.Records
	if !unchanged {
		return "", errors.New("the protected Hardline worktree changed during field-outfit inspection")
	}

	result := overrideRinFieldOutfitInspectResult{
		SchemaVersion: 1, ArtifactID: jobID,
		Branch: overrideRinWardrobeBranch, HeadSHA: overrideRinFieldOutfitExpectedHead,
		ReadOnlySource: true, MutationPerformedSource: false,
		ProtectedStatusSHA256Before: before.SHA256, ProtectedStatusSHA256After: after.SHA256,
		ProtectedStatusRecordsBefore: before.Records, ProtectedStatusRecordsAfter: after.Records,
		ProtectedWorktreeUnchanged: true,
		StageID: overrideRinFieldOutfitStageID, StageCreated: stageCreated, StagedFiles: stagedFiles,
		SubtreeMatchedCount: subtreeCount, SubtreeReturnedCount: len(subtreeFiles), SubtreeTruncated: subtreeTruncated, SubtreeFiles: subtreeFiles,
		Blender: blender,
		VisualAcceptance: "not_assessed", AnimationAcceptance: "not_assessed", AAAAcceptance: false,
	}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > overrideRinFieldOutfitBlenderJSONMaxBytes {
		return "", errors.New("Rin field-outfit inspection result exceeded its privacy-safe output bound")
	}
	return string(encoded), nil
}

func overrideRinFieldOutfitStatus(ctx context.Context, gitExecutable, worktree string) (overrideRinFieldOutfitStatusSnapshot, error) {
	cmd := exec.CommandContext(ctx, gitExecutable, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	cmd.Dir = worktree
	cmd.Env = overrideRinWardrobeGitEnvironment()
	configureChildProcess(cmd, false)
	out := &overrideRinFieldOutfitBoundedBuffer{max: overrideRinFieldOutfitStatusMaxBytes}
	cmd.Stdout = out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil || out.overflow {
		return overrideRinFieldOutfitStatusSnapshot{}, errors.New("bounded protected-worktree status inspection failed")
	}
	raw := out.buffer.Bytes()
	digest := sha256.Sum256(raw)
	records := bytes.Count(raw, []byte{0})
	return overrideRinFieldOutfitStatusSnapshot{SHA256: hex.EncodeToString(digest[:]), Records: records}, nil
}

func overrideRinFieldOutfitTrackedSet(ctx context.Context, gitExecutable, worktree string) (map[string]bool, error) {
	output, err := outputOverrideRinWardrobeGit(ctx, gitExecutable, worktree, "ls-files", "-z", "--", filepath.ToSlash(overrideRinFieldOutfitSubtree))
	if err != nil {
		return nil, errors.New("bounded field-outfit tracked-state inspection failed")
	}
	tracked := map[string]bool{}
	for _, raw := range strings.Split(output, "\x00") {
		path := filepath.ToSlash(strings.TrimSpace(raw))
		if path != "" {
			tracked[strings.ToLower(path)] = true
		}
	}
	return tracked, nil
}

func overrideRinFieldOutfitStagePinnedFiles(ctx context.Context, worktree string, tracked map[string]bool) ([]overrideRinFieldOutfitStagedFile, string, bool, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return nil, "", false, errors.New("Workbench recovery cache is unavailable")
	}
	parent := filepath.Join(cacheRoot, "Workbench", "recovery", "rin-field-outfit", "v1")
	stageRoot := filepath.Join(parent, overrideRinFieldOutfitStageID)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return nil, "", false, errors.New("Workbench could not prepare the field-outfit recovery cache")
	}

	if info, statErr := os.Lstat(stageRoot); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, "", false, errors.New("existing field-outfit stage is not a regular directory")
		}
		files, err := overrideRinFieldOutfitVerifyStage(ctx, worktree, stageRoot, tracked)
		return files, stageRoot, false, err
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, "", false, errors.New("field-outfit stage could not be inspected")
	}

	temp, err := os.MkdirTemp(parent, ".creating-field-outfit-")
	if err != nil {
		return nil, "", false, errors.New("Workbench could not create a field-outfit staging directory")
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(temp)
		}
	}()
	for _, pin := range overrideRinFieldOutfitPinnedFiles {
		if err := ctx.Err(); err != nil {
			return nil, "", false, errors.New("field-outfit staging was cancelled")
		}
		source, err := overrideRinFieldOutfitPinnedSourcePath(worktree, pin)
		if err != nil {
			return nil, "", false, err
		}
		if tracked[strings.ToLower(pin.Path)] {
			return nil, "", false, errors.New("a sealed field-outfit source unexpectedly became tracked")
		}
		if err := overrideRinFieldOutfitVerifyFile(source, pin); err != nil {
			return nil, "", false, err
		}
		relative := strings.TrimPrefix(pin.Path, overrideRinFieldOutfitSubtree+"/")
		destination := filepath.Join(temp, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return nil, "", false, errors.New("Workbench could not prepare a field-outfit staged path")
		}
		if err := overrideRinFieldOutfitCopyExact(source, destination, pin); err != nil {
			return nil, "", false, err
		}
		if err := overrideRinFieldOutfitVerifyFile(source, pin); err != nil {
			return nil, "", false, errors.New("a protected field-outfit source changed while being copied")
		}
	}
	if err := os.Rename(temp, stageRoot); err != nil {
		if _, statErr := os.Stat(stageRoot); statErr != nil {
			return nil, "", false, errors.New("Workbench could not publish the field-outfit recovery stage")
		}
		files, verifyErr := overrideRinFieldOutfitVerifyStage(ctx, worktree, stageRoot, tracked)
		return files, stageRoot, false, verifyErr
	}
	keep = true
	files, err := overrideRinFieldOutfitVerifyStage(ctx, worktree, stageRoot, tracked)
	return files, stageRoot, true, err
}

func overrideRinFieldOutfitVerifyStage(ctx context.Context, worktree, stageRoot string, tracked map[string]bool) ([]overrideRinFieldOutfitStagedFile, error) {
	files := make([]overrideRinFieldOutfitStagedFile, 0, len(overrideRinFieldOutfitPinnedFiles))
	for _, pin := range overrideRinFieldOutfitPinnedFiles {
		if err := ctx.Err(); err != nil {
			return nil, errors.New("field-outfit stage verification was cancelled")
		}
		source, err := overrideRinFieldOutfitPinnedSourcePath(worktree, pin)
		if err != nil {
			return nil, err
		}
		if tracked[strings.ToLower(pin.Path)] {
			return nil, errors.New("a sealed field-outfit source unexpectedly became tracked")
		}
		if err := overrideRinFieldOutfitVerifyFile(source, pin); err != nil {
			return nil, err
		}
		relative := strings.TrimPrefix(pin.Path, overrideRinFieldOutfitSubtree+"/")
		destination := filepath.Join(stageRoot, filepath.FromSlash(relative))
		if err := overrideRinFieldOutfitVerifyFile(destination, pin); err != nil {
			return nil, errors.New("existing field-outfit stage does not match the sealed source bytes")
		}
		files = append(files, overrideRinFieldOutfitStagedFile{Role: pin.Role, Path: pin.Path, Bytes: pin.Bytes, SHA256: pin.SHA256, Tracked: false, Staged: true})
	}
	return files, nil
}

func overrideRinFieldOutfitPinnedSourcePath(worktree string, pin overrideRinFieldOutfitPinnedFile) (string, error) {
	source := filepath.Clean(filepath.Join(worktree, filepath.FromSlash(pin.Path)))
	relative, err := filepath.Rel(worktree, source)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", errors.New("sealed field-outfit path escaped the protected worktree")
	}
	info, err := os.Lstat(source)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", errors.New("a sealed field-outfit source is unavailable or aliased")
	}
	resolved, err := filepath.EvalSymlinks(source)
	if err != nil || !strings.EqualFold(filepath.Clean(resolved), source) {
		return "", errors.New("a sealed field-outfit source resolves through an alias")
	}
	return source, nil
}

func overrideRinFieldOutfitVerifyFile(path string, pin overrideRinFieldOutfitPinnedFile) error {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() != pin.Bytes {
		return errors.New("a sealed field-outfit file size or type changed")
	}
	digest, size, err := FileSHA256(path, pin.Bytes)
	if err != nil || size != pin.Bytes || !strings.EqualFold(digest, pin.SHA256) {
		return errors.New("a sealed field-outfit file hash changed")
	}
	return nil
}

func overrideRinFieldOutfitCopyExact(source, destination string, pin overrideRinFieldOutfitPinnedFile) error {
	in, err := os.Open(source)
	if err != nil {
		return errors.New("a sealed field-outfit source could not be opened")
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return errors.New("a field-outfit staged copy could not be created")
	}
	n, copyErr := io.Copy(out, io.LimitReader(in, pin.Bytes+1))
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil || n != pin.Bytes {
		return errors.New("a field-outfit staged copy was not byte-identical in size")
	}
	if err := overrideRinFieldOutfitVerifyFile(destination, pin); err != nil {
		return errors.New("a field-outfit staged copy failed byte verification")
	}
	return nil
}

func overrideRinFieldOutfitReverifyPinned(ctx context.Context, worktree string, tracked map[string]bool) error {
	for _, pin := range overrideRinFieldOutfitPinnedFiles {
		if err := ctx.Err(); err != nil {
			return errors.New("field-outfit source reverification was cancelled")
		}
		if tracked[strings.ToLower(pin.Path)] {
			return errors.New("a sealed field-outfit source unexpectedly became tracked")
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

func overrideRinFieldOutfitInventorySubtree(ctx context.Context, worktree string, tracked map[string]bool) ([]overrideRinFieldOutfitSubtreeFile, int, bool, error) {
	subtree := filepath.Join(worktree, filepath.FromSlash(overrideRinFieldOutfitSubtree))
	info, err := os.Lstat(subtree)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, 0, false, errors.New("field-outfit source subtree is unavailable or aliased")
	}
	files := make([]overrideRinFieldOutfitSubtreeFile, 0, 128)
	matched := 0
	entries := 0
	truncated := false
	err = filepath.WalkDir(subtree, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		entries++
		if entries > overrideRinFieldOutfitSubtreeMaxEntries {
			return errors.New("field-outfit source subtree exceeds its bounded scan size")
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !overrideRinWardrobeExtensions[ext] {
			return nil
		}
		itemInfo, err := entry.Info()
		if err != nil || !itemInfo.Mode().IsRegular() || itemInfo.Size() < 0 {
			return nil
		}
		relative, err := filepath.Rel(worktree, path)
		if err != nil {
			return nil
		}
		relative = filepath.ToSlash(relative)
		matched++
		if len(files) >= overrideRinFieldOutfitSubtreeMaxReturned {
			truncated = true
			return nil
		}
		files = append(files, overrideRinFieldOutfitSubtreeFile{Path: relative, Extension: ext, Bytes: itemInfo.Size(), Tracked: tracked[strings.ToLower(relative)]})
		return nil
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, 0, false, errors.New("field-outfit source subtree scan was cancelled")
		}
		return nil, 0, false, err
	}
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path) })
	return files, matched, truncated, nil
}

const overrideRinFieldOutfitBlenderInspector = `import bpy, json, os, sys

def clean(v, n=160):
    s = ''.join('-' if ord(c) < 32 or ord(c) == 127 else c for c in str(v))
    return s[:n]

def names(seq, n=32):
    return [clean(x.name) for x in list(seq)[:n]]

def basename(v):
    return clean(os.path.basename(v or ''))

argv = sys.argv
if '--' not in argv or len(argv[argv.index('--')+1:]) != 1:
    raise RuntimeError('sealed inspector requires one internal result path')
out_path = argv[argv.index('--')+1]
objects = []
truncated = len(bpy.data.objects) > 128
for obj in list(bpy.data.objects)[:128]:
    rec = {'name': clean(obj.name), 'type': clean(obj.type, 32)}
    if obj.parent: rec['parent'] = clean(obj.parent.name)
    rec['dimensions'] = [round(float(x), 6) for x in obj.dimensions]
    rec['modifiers'] = [clean(x.type, 48) for x in list(obj.modifiers)[:32]]
    rec['materials'] = [clean(x.name) for x in list(getattr(obj.data, 'materials', []) or [])[:32] if x]
    if getattr(obj.data, 'shape_keys', None): rec['shape_keys'] = names(obj.data.shape_keys.key_blocks)
    if obj.type == 'MESH':
        rec['vertices'] = len(obj.data.vertices); rec['polygons'] = len(obj.data.polygons)
    if obj.type == 'ARMATURE': rec['bones'] = len(obj.data.bones)
    objects.append(rec)
materials = [clean(x.name) for x in list(bpy.data.materials)[:256]]
images = [basename(x.filepath) for x in list(bpy.data.images)[:256]]
libraries = [basename(x.filepath) for x in list(bpy.data.libraries)[:256]]
actions = [{'name': clean(x.name), 'frame_start': round(float(x.frame_range[0]), 3), 'frame_end': round(float(x.frame_range[1]), 3)} for x in list(bpy.data.actions)[:256]]
search = ' '.join([x['name'] for x in objects] + materials + images + [x['name'] for x in actions]).lower()
flags = {
 'rin': 'rin' in search, 'clash': 'clash' in search,
 'shirt': ('shirt' in search or 'tshirt' in search or 'tee' in search),
 'cargo': 'cargo' in search,
 'trousers': ('trouser' in search or 'pants' in search or 'pant' in search),
 'boots': ('boot' in search or 'combat' in search),
 'field': 'field' in search, 'graphic': 'graphic' in search, 'tuck': 'tuck' in search,
}
payload = {'schema_version':1, 'blender_version':clean(bpy.app.version_string,64), 'object_count':len(bpy.data.objects), 'objects':objects, 'materials':materials, 'images':images, 'libraries':libraries, 'actions':actions, 'semantic_flags':flags, 'truncated':truncated}
with open(out_path, 'w', encoding='utf-8') as f:
    json.dump(payload, f, separators=(',', ':'), ensure_ascii=True)
`

func overrideRinFieldOutfitInspectBlend(ctx context.Context, executable, stageRoot string) (overrideRinFieldOutfitBlenderInspection, error) {
	var zero overrideRinFieldOutfitBlenderInspection
	pin, ok := overrideRinFieldOutfitPinnedByPath("SourceAssets/RinWardrobe/AST_RL_CHR_RIN_FIELD_OUTFIT.production.blend")
	if !ok {
		return zero, errors.New("integrated field-outfit pin is unavailable")
	}
	blendPath := filepath.Join(stageRoot, filepath.Base(filepath.FromSlash(pin.Path)))
	if err := overrideRinFieldOutfitVerifyFile(blendPath, pin); err != nil {
		return zero, errors.New("staged integrated field-outfit blend failed verification")
	}
	scratch, err := os.MkdirTemp("", "workbench-rin-field-outfit-inspect-")
	if err != nil {
		return zero, errors.New("Workbench could not create the Blender inspection scratch directory")
	}
	defer os.RemoveAll(scratch)
	scriptPath := filepath.Join(scratch, "inspect.py")
	resultPath := filepath.Join(scratch, "inspection.json")
	if err := os.WriteFile(scriptPath, []byte(overrideRinFieldOutfitBlenderInspector), 0o600); err != nil {
		return zero, errors.New("Workbench could not write the fixed Blender inspector")
	}
	blenderCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(blenderCtx, executable, "--background", "--factory-startup", "--disable-autoexec", blendPath, "--python", scriptPath, "--", resultPath)
	configureChildProcess(cmd, false)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		if blenderCtx.Err() != nil {
			return zero, errors.New("sealed Blender field-outfit inspection timed out")
		}
		return zero, errors.New("sealed Blender field-outfit inspection failed")
	}
	raw, err := os.ReadFile(resultPath)
	if err != nil || len(raw) == 0 || len(raw) > overrideRinFieldOutfitBlenderJSONMaxBytes {
		return zero, errors.New("sealed Blender field-outfit inspection produced an invalid bounded result")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var inspection overrideRinFieldOutfitBlenderInspection
	if err := decoder.Decode(&inspection); err != nil {
		return zero, errors.New("sealed Blender field-outfit inspection JSON was invalid")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return zero, errors.New("sealed Blender field-outfit inspection JSON had trailing content")
	}
	version, err := runBlenderVersion(ctx, executable)
	if err != nil {
		return zero, errors.New("Blender version could not be independently verified")
	}
	inspection.BlenderVersion = version
	if inspection.SchemaVersion != 1 || inspection.ObjectCount < 0 || inspection.SemanticFlags == nil {
		return zero, errors.New("sealed Blender field-outfit inspection result was incomplete")
	}
	return inspection, nil
}
