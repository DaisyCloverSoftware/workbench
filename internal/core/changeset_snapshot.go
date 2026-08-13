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

func SnapshotChangeset(ctx context.Context, project string) (ChangesetSnapshot, error) {
	first, err := InspectChangeset(ctx, project)
	if err != nil {
		return ChangesetSnapshot{}, err
	}
	firstHash, err := fingerprintChangeset(first)
	if err != nil {
		return ChangesetSnapshot{}, err
	}
	second, err := InspectChangeset(ctx, project)
	if err != nil {
		return ChangesetSnapshot{}, err
	}
	if !sameChangesetShape(first, second) {
		return ChangesetSnapshot{}, errors.New("changeset changed during inspection")
	}
	secondHash, err := fingerprintChangeset(second)
	if err != nil {
		return ChangesetSnapshot{}, err
	}
	if firstHash != secondHash {
		return ChangesetSnapshot{}, errors.New("changed-file content changed during inspection")
	}
	return ChangesetSnapshot{Inspection: second, Fingerprint: secondHash}, nil
}

func sameChangesetShape(a, b ChangesetInspection) bool {
	return a.Project == b.Project && a.BaseRevision == b.BaseRevision && a.Clean == b.Clean && a.Safe == b.Safe && a.Diff == b.Diff && reflect.DeepEqual(a.Files, b.Files) && reflect.DeepEqual(a.Untracked, b.Untracked)
}

func fingerprintChangeset(in ChangesetInspection) (string, error) {
	if in.Project == "" || in.BaseRevision == "" || !in.Safe {
		return "", errors.New("changeset inspection is incomplete or unsafe")
	}
	h := sha256.New()
	writeField := func(value string) {
		_, _ = h.Write([]byte(value))
		_, _ = h.Write([]byte{0})
	}
	writeField(in.BaseRevision)
	writeField(in.Diff)
	for _, rel := range in.Files {
		writeField(rel)
		path := filepath.Join(in.Project, filepath.FromSlash(rel))
		st, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			writeField("deleted")
			continue
		}
		if err != nil {
			return "", err
		}
		if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
			return "", fmt.Errorf("changeset path is no longer regular: %s", rel)
		}
		if st.Size() > maxChangedFileBytes {
			return "", fmt.Errorf("changed file is too large: %s", rel)
		}
		writeField(fmt.Sprintf("%#o", st.Mode().Perm()))
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		_, _ = h.Write(b)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
