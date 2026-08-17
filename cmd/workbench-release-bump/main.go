package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type releaseRequest struct {
	Version string   `json:"version"`
	Date    string   `json:"date"`
	Notes   []string `json:"notes"`
}

type preparedFile struct {
	path string
	mode os.FileMode
	data []byte
}

var (
	semverPattern  = regexp.MustCompile(`^([0-9]+)\.([0-9]+)\.([0-9]+)$`)
	datePattern    = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	versionPattern = regexp.MustCompile(`const Version = "([0-9]+\.[0-9]+\.[0-9]+)"`)
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: workbench-release-bump <release-request.json>")
		os.Exit(2)
	}
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	req, err := loadReleaseRequest(os.Args[1])
	if err != nil {
		fatal(err)
	}
	if err := prepareRelease(root, req); err != nil {
		fatal(err)
	}
	fmt.Printf("Prepared Workbench %s release metadata.\n", req.Version)
}

func loadReleaseRequest(path string) (releaseRequest, error) {
	var req releaseRequest
	b, err := os.ReadFile(path)
	if err != nil {
		return req, err
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return req, fmt.Errorf("invalid release request: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err == nil {
		return req, errors.New("release request must contain exactly one JSON object")
	}
	return req, validateRequest(req)
}

func validateRequest(req releaseRequest) error {
	if !semverPattern.MatchString(strings.TrimSpace(req.Version)) {
		return errors.New("release version must be x.y.z")
	}
	if !datePattern.MatchString(strings.TrimSpace(req.Date)) {
		return errors.New("release date must be YYYY-MM-DD")
	}
	if len(req.Notes) == 0 || len(req.Notes) > 30 {
		return errors.New("release request must contain 1-30 notes")
	}
	for _, raw := range req.Notes {
		note := strings.TrimSpace(raw)
		if note == "" || len(note) > 600 || strings.ContainsAny(note, "\r\n") {
			return errors.New("release notes must be non-empty single lines up to 600 characters")
		}
	}
	return nil
}

func prepareRelease(root string, req releaseRequest) error {
	if err := validateRequest(req); err != nil {
		return err
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	versionPath := filepath.Join(root, "internal", "core", "version.go")
	versionBytes, err := os.ReadFile(versionPath)
	if err != nil {
		return fmt.Errorf("read canonical version: %w", err)
	}
	match := versionPattern.FindSubmatch(versionBytes)
	if len(match) != 2 {
		return errors.New("canonical Workbench version could not be resolved")
	}
	current := string(match[1])
	if !versionGreater(req.Version, current) {
		return fmt.Errorf("release version %s must be greater than current %s", req.Version, current)
	}

	type replacement struct {
		path string
		old  string
		new  string
	}
	replacements := []replacement{
		{filepath.Join(".codex-plugin", "plugin.json"), `"version": "` + current + `"`, `"version": "` + req.Version + `"`},
		{filepath.Join("cmd", "workbench", "main_windows.go"), `const appVersion = "` + current + `"`, `const appVersion = "` + req.Version + `"`},
		{filepath.Join("cmd", "workbench-runner", "main.go"), `const runnerVersion = "` + current + `"`, `const runnerVersion = "` + req.Version + `"`},
		{filepath.Join("cmd", "workbench-server", "main.go"), `const serverVersion = "` + current + `"`, `const serverVersion = "` + req.Version + `"`},
		{filepath.Join("cmd", "workbench-relay", "main.go"), `const relayVersion = "` + current + `"`, `const relayVersion = "` + req.Version + `"`},
		{filepath.Join("internal", "core", "version.go"), `const Version = "` + current + `"`, `const Version = "` + req.Version + `"`},
		{filepath.Join("internal", "mcp", "server.go"), `"version": "` + current + `"`, `"version": "` + req.Version + `"`},
	}

	prepared := make([]preparedFile, 0, len(replacements)+1)
	for _, repl := range replacements {
		full := filepath.Join(root, repl.path)
		b, err := os.ReadFile(full)
		if err != nil {
			return fmt.Errorf("read %s: %w", repl.path, err)
		}
		if strings.Count(string(b), repl.old) != 1 {
			return fmt.Errorf("%s does not contain exactly one expected %s version token", repl.path, current)
		}
		info, err := os.Stat(full)
		if err != nil {
			return err
		}
		updated := strings.Replace(string(b), repl.old, repl.new, 1)
		prepared = append(prepared, preparedFile{path: full, mode: info.Mode().Perm(), data: []byte(updated)})
	}

	changelogPath := filepath.Join(root, "CHANGELOG.md")
	changelog, err := os.ReadFile(changelogPath)
	if err != nil {
		return fmt.Errorf("read changelog: %w", err)
	}
	if strings.Contains(string(changelog), "## "+req.Version) {
		return fmt.Errorf("CHANGELOG already contains Workbench %s", req.Version)
	}
	const changelogPrefix = "# Changelog\n\n"
	if !strings.HasPrefix(string(changelog), changelogPrefix) {
		return errors.New("CHANGELOG.md has an unexpected header")
	}
	var entry strings.Builder
	fmt.Fprintf(&entry, "## %s — %s\n\n", req.Version, req.Date)
	for _, note := range req.Notes {
		fmt.Fprintf(&entry, "- %s\n", strings.TrimSpace(note))
	}
	entry.WriteString("\n")
	changelogUpdated := changelogPrefix + entry.String() + strings.TrimPrefix(string(changelog), changelogPrefix)
	info, err := os.Stat(changelogPath)
	if err != nil {
		return err
	}
	prepared = append(prepared, preparedFile{path: changelogPath, mode: info.Mode().Perm(), data: []byte(changelogUpdated)})

	// All input files are validated before the first write so a malformed release
	// request cannot leave a partially bumped checkout on a long-lived host.
	for _, file := range prepared {
		if err := writeAtomic(file.path, file.data, file.mode); err != nil {
			return err
		}
	}
	return nil
}

func versionGreater(next, current string) bool {
	a := semverPattern.FindStringSubmatch(next)
	b := semverPattern.FindStringSubmatch(current)
	if len(a) != 4 || len(b) != 4 {
		return false
	}
	for i := 1; i <= 3; i++ {
		av, _ := strconv.Atoi(a[i])
		bv, _ := strconv.Atoi(b[i])
		if av != bv {
			return av > bv
		}
	}
	return false
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".workbench-release-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "workbench-release-bump:", err)
	os.Exit(1)
}
