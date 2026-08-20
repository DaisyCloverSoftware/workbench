package core

import "testing"

func TestWorkItemFromHostJobShowsRealWindowsLocation(t *testing.T) {
	job := HostJob{
		ID:        "hostjob_test1234",
		HostID:    "windows_test1234",
		Spec:      HostJobSpec{Tool: HostBridgeToolUnreal, Operation: HostBridgeOperationVersion},
		Status:    "claimed",
		CreatedAt: "2026-08-20T07:00:00Z",
		UpdatedAt: "2026-08-20T07:01:00Z",
		ClaimedAt: "2026-08-20T07:00:30Z",
	}
	host := HostBridgeHost{HostID: job.HostID, Label: "Workstation", Online: true}
	item := WorkItemFromHostJob(job, host)
	if item.Location.Lane != WorkLaneWindowsHost || item.Location.Machine != "Workstation" || item.Location.Tool != "Unreal" {
		t.Fatalf("location = %#v", item.Location)
	}
	if item.State != WorkItemRunning || item.Progress.Kind != WorkProgressIndeterminate {
		t.Fatalf("running host projection = %#v", item)
	}
	if item.CanReprioritize || item.CanCancel || item.CanMove {
		t.Fatalf("host controls must stay disabled until implemented safely: %#v", item)
	}
}

func TestWorkItemFromHostJobReportsOfflineQueueBlocker(t *testing.T) {
	job := HostJob{
		ID:        "hostjob_test5678",
		HostID:    "windows_test5678",
		Spec:      HostJobSpec{Tool: HostBridgeToolBlender, Operation: HostBridgeOperationVersion},
		Status:    "queued",
		CreatedAt: "2026-08-20T07:00:00Z",
		UpdatedAt: "2026-08-20T07:00:00Z",
	}
	item := WorkItemFromHostJob(job, HostBridgeHost{HostID: job.HostID, Label: "Offline PC", Online: false})
	if item.State != WorkItemQueued || item.Blocker != "Target Windows host is offline" {
		t.Fatalf("offline queued projection = %#v", item)
	}
}

func TestActiveWorkItemsFromHostJobsExcludesHistory(t *testing.T) {
	jobs := []HostJob{
		{ID: "hostjob_queue123", HostID: "windows_host123", Status: "queued"},
		{ID: "hostjob_done1234", HostID: "windows_host123", Status: "completed"},
		{ID: "hostjob_fail1234", HostID: "windows_host123", Status: "failed"},
	}
	items := ActiveWorkItemsFromHostJobs(jobs, []HostBridgeHost{{HostID: "windows_host123", Label: "PC", Online: true}})
	if len(items) != 1 || items[0].ID != "hostjob_queue123" {
		t.Fatalf("active host items = %#v", items)
	}
}
