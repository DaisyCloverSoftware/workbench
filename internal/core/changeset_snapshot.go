package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

type ChangesetSnapshot struct {
	Inspection  ChangesetInspection `json:"inspection"`
	Fingerprint string              `json:"fingerprint"`
}

// SnapshotChangeset returns a stable changeset inspection plus a digest of the
// exact changed-file contents and modes. It samples twice and refuses a moving
// target so a later deterministic publisher can detect edits made between
// inspection and publication.
func SnapshotChangeset(ctx context.Context, project string) (ChangesetSnapshot, error) {
	first, err := InspectChangeset(ctx, project)
	if err != nil {
		return ChangesetSnapshot{}, err
	}
	firstDigest, err := fingerprintChangeset(first)
	if err != nil {
		return ChangesetSnapshot{}, err
	}
	second, err := InspectChangeset(ctx, project)
	if err != nil {
		return ChangesetSnapshot{}, err
	}
	if !sameChangesetShape(first, second) {
		return ChangesetSnapshot{}, errors.New("changeset changed while Workbench was inspecting it; inspect again")
	}
	secondDigest, err := fingerprintChangeset(second)
	if err != nil {
		return ChangesetSnapshot{}, err
	}
	if firstDigest != secondDigest {
		return ChangesetSnapshot{}, errors.New("changed-file contents changed while Workbench was inspecting them; inspect again")
	}
	return ChangesetSnapshot{Inspection: second, Fingerprint: secondDigest}, nil
}

func sameChangesetShape(a, b ChangesetInspection) bool {
	return a.Project == b.Project &&
		a.BaseRevision == b.BaseRevision &&
		a.Clean == b.Clean &&
		a.Safe == b.Safe &&
		a.Diff == b.Diff &&
		reflect.DeepEqual(a.Files, b.Files) &&
		reflect.DeepEqual(a.Untracked, b.Untracked)
}

func fingerprintChangeset(in ChangesetInspection) (string, error) {
	if in.Project == "" || in.BaseRevision == "" || !in.Safe {
		return "", errors.New("changeset inspection is incomplete or unsafe")
	}
	h := sha256.New()
	writeDigestField := func(value string) {
		_, _ = h.Write([]byte(value))
		_, _ = h.Write([]byte{0})
	}
	writeDigestField(in.BaseRevision)
	writeDigestField(in.Diff)
	for _, rel := range in.Files {
		writeDigestField(rel)
		path := filepath.Join(in.Project, filepath.FromSlash(rel))
		st, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			writeDigestField("deleted")
			continue
		}
		if err != nil {
			return "", err
		}
		if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
			return "", fmt.Errorf("changeset path is no longer a regular file: %s", rel)
		}
		if st.Size() > maxChangedFileBytes {
			return "", fmt.Errorf("changed file exceeds %d bytes: %s", maxChangedFileBytes, rel)
		}
		writeDigestField(fmt.Sprintf("%#o", st.Mode().Perm()))
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		_, _ = h.Write(b)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
