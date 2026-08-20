package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ListHostBridgeJobs exposes the durable host queue for dashboard/status views.
// It does not claim, mutate, retry or otherwise influence execution.
func ListHostBridgeJobs() ([]HostJob, error) {
	var jobs []HostJob
	err := withHostBridgeLock(func(root string) error {
		entries, err := os.ReadDir(filepath.Join(root, "jobs"))
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			var job HostJob
			if err := readHostBridgeJSON(filepath.Join(root, "jobs", entry.Name()), &job); err != nil {
				continue
			}
			jobs = append(jobs, job)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		left, leftErr := time.Parse(time.RFC3339Nano, jobs[i].CreatedAt)
		right, rightErr := time.Parse(time.RFC3339Nano, jobs[j].CreatedAt)
		if leftErr == nil && rightErr == nil && !left.Equal(right) {
			return left.After(right)
		}
		return jobs[i].ID > jobs[j].ID
	})
	return jobs, nil
}
