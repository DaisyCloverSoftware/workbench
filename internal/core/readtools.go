package core

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxReadFileBytes   = 2 << 20 // 2 MiB source-file ceiling
	maxReadOutputBytes = 256 << 10
	maxSearchFileBytes = 1 << 20
)

type SearchHit struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

func projectRoot(project string) (string, error) {
	root, err := filepath.Abs(strings.TrimSpace(project))
	if err != nil {
		return "", err
	}
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return "", fmt.Errorf("project path is not a directory: %s", root)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func resolveProjectReadPath(project, relative string) (string, string, error) {
	root, err := projectRoot(project)
	if err != nil {
		return "", "", err
	}
	relative = strings.TrimSpace(relative)
	if relative == "" {
		return root, ".", nil
	}
	normalized := filepath.FromSlash(strings.ReplaceAll(relative, "\\", "/"))
	if filepath.IsAbs(normalized) {
		return "", "", errors.New("file path must be relative to the active project")
	}
	clean := filepath.Clean(normalized)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", "", errors.New("file path escapes the active project")
	}
	candidate := filepath.Join(root, clean)
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", "", err
	}
	if !withinProjectRoot(root, resolved) {
		return "", "", errors.New("resolved file path escapes the active project")
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", "", err
	}
	return resolved, filepath.ToSlash(rel), nil
}

func withinProjectRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func sensitiveProjectPath(rel string) bool {
	rel = strings.ToLower(filepath.ToSlash(rel))
	parts := strings.Split(rel, "/")
	for _, p := range parts {
		if p == ".ssh" || p == ".gnupg" || p == ".aws" || p == ".azure" || p == ".kube" {
			return true
		}
	}
	base := filepath.Base(rel)
	if base == ".env" || strings.HasPrefix(base, ".env.") || base == "credentials" || base == "credentials.json" || base == "secrets.json" {
		return true
	}
	for _, suffix := range []string{".pem", ".key", ".p12", ".pfx", ".jks", ".keystore"} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	for _, name := range []string{"id_rsa", "id_dsa", "id_ecdsa", "id_ed25519"} {
		if base == name || strings.HasPrefix(base, name+".") {
			return true
		}
	}
	return false
}

func skippedTreeDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".hg", ".svn", "node_modules", "vendor", "dist", "build", ".next", ".cache", "coverage", "target", "bin", "obj":
		return true
	default:
		return false
	}
}

func ListProjectFiles(project, subdir string, limit int) ([]string, error) {
	if limit <= 0 || limit > 1000 {
		limit = 300
	}
	root, err := projectRoot(project)
	if err != nil {
		return nil, err
	}
	start := root
	if strings.TrimSpace(subdir) != "" && strings.TrimSpace(subdir) != "." {
		var rel string
		start, rel, err = resolveProjectReadPath(project, subdir)
		if err != nil {
			return nil, err
		}
		if sensitiveProjectPath(rel) {
			return nil, errors.New("Workbench will not enumerate a sensitive credential directory")
		}
	}
	st, err := os.Stat(start)
	if err != nil || !st.IsDir() {
		return nil, errors.New("list path is not a directory")
	}

	out := make([]string, 0, limit)
	err = filepath.WalkDir(start, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path == start {
			return nil
		}
		if d.IsDir() {
			if skippedTreeDir(d.Name()) || sensitiveProjectPath(mustRel(root, path)) {
				return filepath.SkipDir
			}
			return nil
		}
		if len(out) >= limit {
			return fs.SkipAll
		}
		rel := mustRel(root, path)
		if rel == "" || sensitiveProjectPath(rel) {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr == nil && info.Mode().IsRegular() {
			out = append(out, rel)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

func ReadProjectFile(project, relative string, startLine, endLine int) (string, error) {
	path, rel, err := resolveProjectReadPath(project, relative)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return "", errors.New("read_file requires a file path")
	}
	if sensitiveProjectPath(rel) {
		return "", errors.New("Workbench refused to expose a sensitive credential file to the model")
	}
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "", errors.New("path is not a regular file")
	}
	if st.Size() > maxReadFileBytes {
		return "", fmt.Errorf("file is too large for model-facing read (%d bytes)", st.Size())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if bytes.IndexByte(b, 0) >= 0 {
		return "", errors.New("Workbench refused to expose a binary file to the model")
	}
	text := string(b)
	if LooksSecret(text) {
		return "", errors.New("Workbench detected probable secret material in this file and refused to expose it to the model")
	}
	if startLine <= 0 {
		startLine = 1
	}
	lines := strings.Split(text, "\n")
	if endLine <= 0 || endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine || startLine > len(lines) {
		return "", errors.New("requested line range is outside the file")
	}
	var out strings.Builder
	for i := startLine; i <= endLine; i++ {
		fmt.Fprintf(&out, "%6d | %s", i, lines[i-1])
		if i < endLine {
			out.WriteByte('\n')
		}
		if out.Len() > maxReadOutputBytes {
			out.WriteString("\n… output truncated by Workbench …")
			break
		}
	}
	return out.String(), nil
}

func SearchProjectText(project, query, subdir string, limit int) ([]SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("search query is empty")
	}
	if len(query) > 500 {
		return nil, errors.New("search query is too long")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	root, err := projectRoot(project)
	if err != nil {
		return nil, err
	}
	start := root
	if strings.TrimSpace(subdir) != "" && strings.TrimSpace(subdir) != "." {
		var rel string
		start, rel, err = resolveProjectReadPath(project, subdir)
		if err != nil {
			return nil, err
		}
		if sensitiveProjectPath(rel) {
			return nil, errors.New("Workbench will not search a sensitive credential directory")
		}
	}
	needle := strings.ToLower(query)
	hits := make([]SearchHit, 0, limit)
	err = filepath.WalkDir(start, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if path != start && (skippedTreeDir(d.Name()) || sensitiveProjectPath(mustRel(root, path))) {
				return filepath.SkipDir
			}
			return nil
		}
		if len(hits) >= limit {
			return fs.SkipAll
		}
		rel := mustRel(root, path)
		if rel == "" || sensitiveProjectPath(rel) {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil || !info.Mode().IsRegular() || info.Size() > maxSearchFileBytes {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil || bytes.IndexByte(b, 0) >= 0 || LooksSecret(string(b)) {
			return nil
		}
		scanner := bufio.NewScanner(bytes.NewReader(b))
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			if strings.Contains(strings.ToLower(line), needle) {
				if len(line) > 1000 {
					line = line[:1000] + "…"
				}
				hits = append(hits, SearchHit{Path: rel, Line: lineNo, Text: line})
				if len(hits) >= limit {
					return fs.SkipAll
				}
			}
		}
		return nil
	})
	return hits, err
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}
