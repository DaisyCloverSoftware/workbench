//go:build windows

package desktop

import (
	"context"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
	"github.com/DaisyCloverSoftware/workbench/internal/mcp"
)

type shellPage int

const (
	pageWork shellPage = iota
	pageSettings
)

const (
	idNavWork = 3001
	idNavSettings = 3002
	idBrand = 3003
	idGlobalStatus = 3004

	idProjectsLabel = 3100
	idProjectList = 3101
	idAddProject = 3102
	idProjectName = 3103
	idRenameProject = 3104
	idPinProject = 3105
	idRemoveProject = 3106
	idActiveProject = 3107
	idActivePath = 3108
	idSummary = 3109
	idIntentLabel = 3110
	idIntent = 3111
	idDelegate = 3112
	idTasksLabel = 3113
	idTaskList = 3114
	idTaskStatus = 3115
	idNextAction = 3116
	idReportLabel = 3117
	idReport = 3118
	idCancelTask = 3119
	idAttentionLabel = 3120
	idAnswer = 3121
	idResumeTask = 3122
	idOpenReview = 3123
	idRetryReview = 3124
	idCopyBranch = 3125
	idNotesLabel = 3126
	idNotes = 3127
	idSaveNotes = 3128

	idSettingsTitle = 3200
	idProvidersLabel = 3201
	idProviderList = 3202
	idConnectProvider = 3203
	idRescanProviders = 3204
	idProtectWork = 3205
	idAllowMetered = 3206
	idMCPLabel = 3207
	idMCPStatus = 3208
	idCopyMCP = 3209
	idRunnerLabel = 3210
	idRunnerHost = 3211
	idHarnessLabel = 3212
	idHarnessCommand = 3213
	idNotifyLabel = 3214
	idNotifyCommand = 3215
	idSaveRouting = 3216
	idReviewLabel = 3217
	idPublishReviews = 3218
	idReviewRemote = 3219
	idSaveReviewPolicy = 3220
	idVaultLabel = 3221
	idSecretName = 3222
	idSecretValue = 3223
	idSaveSecret = 3224
	idSecretList = 3225
	idMaintenanceLabel = 3226
	idRunUpdater = 3227
)

type Shell struct {
	hwnd             uintptr
	eng              *core.Engine
	mcp              *mcp.Server
	mcpURL           string
	mcpErr           string
	version          string
	font             uintptr
	backgroundBrush  uintptr
	controls         map[int]uintptr
	page             shellPage
	projectIDs       []string
	taskIDs          []string
	providerIDs      []string
	selectedTaskID   string
	editorProjectID  string
	settingsProjectID string
}

var runningShell *Shell

func Run(version string) error {
	store, err := core.NewStore()
	if err != nil {
		return fmt.Errorf("open Workbench state: %w", err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		return fmt.Errorf("initialise Workbench: %w", err)
	}
	st := eng.State()
	var srv *mcp.Server
	var mcpURL, mcpErr string
	for port := st.Preferences.MCPPort; port < st.Preferences.MCPPort+20; port++ {
		candidate := mcp.New(eng, port, st.Preferences.MCPToken)
		if startErr := candidate.Start(); startErr == nil {
			srv = candidate
			mcpURL = candidate.URL()
			if port != st.Preferences.MCPPort {
				prefs := st.Preferences
				prefs.MCPPort = port
				_ = eng.SavePreferences(prefs)
			}
			break
		} else {
			mcpErr = startErr.Error()
		}
	}
	// Resume local durable tasks only after the desktop has acquired its MCP
	// listener, matching the headless-server recovery ordering.
	_ = eng.ResumeInterruptedTasks()

	shell := &Shell{
		eng:      eng,
		mcp:      srv,
		mcpURL:   mcpURL,
		mcpErr:   mcpErr,
		version:  strings.TrimSpace(version),
		controls: map[int]uintptr{},
		page:     pageWork,
	}
	runningShell = shell
	defer func() { runningShell = nil }()
	return shell.run()
}

func (s *Shell) run() error {
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	className := wstr("DaisyCloverWorkbenchProductionWindow")
	icon, _, _ := user32.NewProc("LoadIconW").Call(0, 32512)
	cursor, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
	s.backgroundBrush, _, _ = procCreateSolidBrush.Call(uintptr(rgb(18, 21, 26)))
	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   syscall.NewCallback(shellWndProc),
		Instance:  instance,
		Icon:      icon,
		Cursor:    cursor,
		Background: s.backgroundBrush,
		ClassName: uintptr(unsafe.Pointer(className)),
		IconSm:    icon,
	}
	if r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return e
	}
	title := "Workbench"
	if s.version != "" {
		title += " " + s.version
	}
	title += " — Autonomous developer workspace"
	titlePtr := wstr(title)
	hwnd, _, e := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(titlePtr)),
		wsOverlappedWindow|wsVisible,
		72, 48, 1500, 920,
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		return e
	}
	s.hwnd = hwnd
	useDark := uint32(1)
	_, _, _ = procDwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&useDark)), unsafe.Sizeof(useDark))
	s.eng.Subscribe(func() {
		if s.hwnd != 0 {
			procPostMessageW.Call(s.hwnd, wmAppRefresh, 0, 0)
		}
	})
	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)

	var message nativeMsg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
	if s.mcp != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = s.mcp.Close(ctx)
		cancel()
	}
	return nil
}

func shellWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	s := runningShell
	if s == nil {
		return defWindowProc(hwnd, message, wParam, lParam)
	}
	switch message {
	case wmCreate:
		s.hwnd = hwnd
		s.createControls()
		s.refresh()
		s.layout()
		return 0
	case wmSize:
		s.layout()
		return 0
	case wmAppRefresh:
		s.refresh()
		return 0
	case wmCommand:
		id := int(uint16(wParam & 0xffff))
		notify := uint16((wParam >> 16) & 0xffff)
		s.handleCommand(id, notify)
		return 0
	case wmCtlColorStatic, wmCtlColorBtn, wmCtlColorEdit, wmCtlColorListBox:
		hdc := wParam
		procSetTextColor.Call(hdc, uintptr(rgb(235, 238, 244)))
		procSetBkColor.Call(hdc, uintptr(rgb(18, 21, 26)))
		return s.backgroundBrush
	case wmClose:
		user32.NewProc("DestroyWindow").Call(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	return defWindowProc(hwnd, message, wParam, lParam)
}

func (s *Shell) createControls() {
	s.font, _, _ = procGetStockObject.Call(defaultGUIFont)
	s.static(idBrand, "WORKBENCH  ·  AUTONOMOUS DEVELOPER WORKSPACE")
	s.static(idGlobalStatus, "")
	s.control(idNavWork, "BUTTON", "Work", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idNavSettings, "BUTTON", "Settings", wsChild|wsVisible|wsTabStop|bsPushButton)

	// Work page.
	s.static(idProjectsLabel, "PROJECTS")
	s.control(idProjectList, "LISTBOX", "", wsChild|wsVisible|wsBorder|wsVScroll|lbsNotify)
	s.control(idAddProject, "BUTTON", "+ Add project", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idProjectName, "EDIT", "", wsChild|wsVisible|wsBorder|wsTabStop|esAutoHScroll)
	s.control(idRenameProject, "BUTTON", "Rename", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idPinProject, "BUTTON", "Pin / unpin", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idRemoveProject, "BUTTON", "Remove", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.static(idActiveProject, "No project selected")
	s.static(idActivePath, "")
	s.static(idSummary, "")
	s.static(idIntentLabel, "What do you want Workbench to finish?")
	s.control(idIntent, "EDIT", "", wsChild|wsVisible|wsBorder|wsTabStop|esMultiline|esAutoVScroll|wsVScroll)
	s.control(idDelegate, "BUTTON", "Delegate autonomously", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.static(idTasksLabel, "TASKS")
	s.control(idTaskList, "LISTBOX", "", wsChild|wsVisible|wsBorder|wsVScroll|lbsNotify)
	s.static(idTaskStatus, "")
	s.static(idNextAction, "")
	s.static(idReportLabel, "RESULT / ACTIVITY")
	s.control(idReport, "EDIT", "", wsChild|wsVisible|wsBorder|esMultiline|esAutoVScroll|wsVScroll|esReadOnly)
	s.control(idCancelTask, "BUTTON", "Cancel", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.static(idAttentionLabel, "Only answer here when this task genuinely needs you")
	s.control(idAnswer, "EDIT", "", wsChild|wsVisible|wsBorder|wsTabStop|esAutoHScroll)
	s.control(idResumeTask, "BUTTON", "Answer + resume", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idOpenReview, "BUTTON", "Open review", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idRetryReview, "BUTTON", "Retry delivery", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idCopyBranch, "BUTTON", "Copy review branch", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.static(idNotesLabel, "PROJECT CONTEXT / NOTES")
	s.control(idNotes, "EDIT", "", wsChild|wsVisible|wsBorder|wsTabStop|esMultiline|esAutoVScroll|wsVScroll)
	s.control(idSaveNotes, "BUTTON", "Save project notes", wsChild|wsVisible|wsTabStop|bsPushButton)

	cueBanner(s.controls[idIntent], "Describe the outcome, not the implementation steps. Workbench will choose the worker and keep going.")
	cueBanner(s.controls[idAnswer], "Your decision or permission")
	cueBanner(s.controls[idProjectName], "Project display name")
	cueBanner(s.controls[idNotes], "Durable project context. Secrets are refused here; store them in Settings → Vault.")

	// Settings page.
	s.static(idSettingsTitle, "SETTINGS  ·  ROUTING, CONNECTIONS, REVIEW POLICY, VAULT & MAINTENANCE")
	s.static(idProvidersLabel, "AI WORKERS")
	s.control(idProviderList, "LISTBOX", "", wsChild|wsVisible|wsBorder|wsVScroll|lbsNotify)
	s.control(idConnectProvider, "BUTTON", "Connect selected", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idRescanProviders, "BUTTON", "Rescan", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idProtectWork, "BUTTON", "Protect scarce Work/Codex usage", wsChild|wsVisible|wsTabStop|bsAutoCheckbox)
	s.control(idAllowMetered, "BUTTON", "Allow metered API fallback", wsChild|wsVisible|wsTabStop|bsAutoCheckbox)
	s.static(idMCPLabel, "CHAT / MCP BRIDGE")
	s.static(idMCPStatus, "")
	s.control(idCopyMCP, "BUTTON", "Copy MCP connection", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.static(idRunnerLabel, "Workbench Runner SSH host")
	s.control(idRunnerHost, "EDIT", "", wsChild|wsVisible|wsBorder|wsTabStop|esAutoHScroll)
	s.static(idHarnessLabel, "Optional custom harness command")
	s.control(idHarnessCommand, "EDIT", "", wsChild|wsVisible|wsBorder|wsTabStop|esAutoHScroll)
	s.static(idNotifyLabel, "Optional human-interrupt command · {message}")
	s.control(idNotifyCommand, "EDIT", "", wsChild|wsVisible|wsBorder|wsTabStop|esAutoHScroll)
	s.control(idSaveRouting, "BUTTON", "Save routing settings", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.static(idReviewLabel, "REVIEW DELIVERY · active project only")
	s.control(idPublishReviews, "BUTTON", "Publish Workbench review branches", wsChild|wsVisible|wsTabStop|bsAutoCheckbox)
	s.control(idReviewRemote, "EDIT", "", wsChild|wsVisible|wsBorder|wsTabStop|esAutoHScroll)
	s.control(idSaveReviewPolicy, "BUTTON", "Save review policy", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.static(idVaultLabel, "ENCRYPTED VAULT · WINDOWS DPAPI · VALUES NEVER SHOWN TO AI")
	s.control(idSecretName, "EDIT", "", wsChild|wsVisible|wsBorder|wsTabStop|esAutoHScroll)
	s.control(idSecretValue, "EDIT", "", wsChild|wsVisible|wsBorder|wsTabStop|esAutoHScroll|esPassword)
	s.control(idSaveSecret, "BUTTON", "Encrypt + store", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idSecretList, "LISTBOX", "", wsChild|wsVisible|wsBorder|wsVScroll)
	s.static(idMaintenanceLabel, "MAINTENANCE")
	s.control(idRunUpdater, "BUTTON", "Check / install verified update", wsChild|wsVisible|wsTabStop|bsPushButton)

	cueBanner(s.controls[idRunnerHost], "user@runner-host")
	cueBanner(s.controls[idHarnessCommand], "advanced adapter: command {project} {prompt}")
	cueBanner(s.controls[idNotifyCommand], "command using {message}")
	cueBanner(s.controls[idReviewRemote], "explicit HTTPS/SSH review remote; no embedded credentials")
	cueBanner(s.controls[idSecretName], "vault name, e.g. prod/ssh")
	cueBanner(s.controls[idSecretValue], "secret value")

	s.applyPageVisibility()
}

func (s *Shell) static(id int, text string) uintptr {
	return s.control(id, "STATIC", text, wsChild|wsVisible)
}

func (s *Shell) control(id int, class, text string, style uintptr) uintptr {
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	classPtr, textPtr := wstr(class), wstr(text)
	exStyle := uintptr(0)
	if class == "EDIT" || class == "LISTBOX" {
		exStyle = wsExClientEdge
	}
	hwnd, _, _ := procCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(classPtr)),
		uintptr(unsafe.Pointer(textPtr)),
		style,
		0, 0, 10, 10,
		s.hwnd,
		uintptr(id),
		instance,
		0,
	)
	s.controls[id] = hwnd
	procSendMessageW.Call(hwnd, wmSetFont, s.font, 1)
	return hwnd
}

func (s *Shell) workControlIDs() []int {
	return []int{
		idProjectsLabel, idProjectList, idAddProject, idProjectName, idRenameProject, idPinProject, idRemoveProject,
		idActiveProject, idActivePath, idSummary, idIntentLabel, idIntent, idDelegate, idTasksLabel, idTaskList,
		idTaskStatus, idNextAction, idReportLabel, idReport, idCancelTask, idAttentionLabel, idAnswer, idResumeTask,
		idOpenReview, idRetryReview, idCopyBranch, idNotesLabel, idNotes, idSaveNotes,
	}
}

func (s *Shell) settingsControlIDs() []int {
	return []int{
		idSettingsTitle, idProvidersLabel, idProviderList, idConnectProvider, idRescanProviders, idProtectWork, idAllowMetered,
		idMCPLabel, idMCPStatus, idCopyMCP, idRunnerLabel, idRunnerHost, idHarnessLabel, idHarnessCommand, idNotifyLabel,
		idNotifyCommand, idSaveRouting, idReviewLabel, idPublishReviews, idReviewRemote, idSaveReviewPolicy,
		idVaultLabel, idSecretName, idSecretValue, idSaveSecret, idSecretList, idMaintenanceLabel, idRunUpdater,
	}
}

func (s *Shell) applyPageVisibility() {
	for _, id := range s.workControlIDs() {
		showWindow(s.controls[id], s.page == pageWork)
	}
	for _, id := range s.settingsControlIDs() {
		showWindow(s.controls[id], s.page == pageSettings)
	}
}

func (s *Shell) layout() {
	if s.hwnd == 0 {
		return
	}
	var r nativeRect
	procGetClientRect.Call(s.hwnd, uintptr(unsafe.Pointer(&r)))
	width := int(r.Right - r.Left)
	height := int(r.Bottom - r.Top)
	if width < 1080 {
		width = 1080
	}
	pad := 16
	header := 70
	navW := 112
	contentX := navW + pad*2
	contentW := width - contentX - pad

	moveWindow(s.controls[idBrand], pad, 14, width-2*pad, 22)
	moveWindow(s.controls[idGlobalStatus], pad, 39, width-2*pad, 20)
	moveWindow(s.controls[idNavWork], pad, header+8, navW, 38)
	moveWindow(s.controls[idNavSettings], pad, header+54, navW, 38)

	if s.page == pageWork {
		s.layoutWork(contentX, header, contentW, height-header-pad)
	} else {
		s.layoutSettings(contentX, header, contentW, height-header-pad)
	}
}

func (s *Shell) layoutWork(x, y, width, height int) {
	left := 270
	right := 320
	gap := 16
	center := width - left - right - gap*2
	if center < 430 {
		center = 430
	}
	xLeft := x
	xCenter := x + left + gap
	xRight := xCenter + center + gap
	top := y + 8

	moveWindow(s.controls[idProjectsLabel], xLeft, top, left, 20)
	moveWindow(s.controls[idProjectList], xLeft, top+24, left, height-250)
	moveWindow(s.controls[idAddProject], xLeft, top+height-218, left, 32)
	moveWindow(s.controls[idProjectName], xLeft, top+height-178, left-88, 28)
	moveWindow(s.controls[idRenameProject], xLeft+left-80, top+height-178, 80, 28)
	moveWindow(s.controls[idPinProject], xLeft, top+height-142, left/2-4, 30)
	moveWindow(s.controls[idRemoveProject], xLeft+left/2+4, top+height-142, left/2-4, 30)

	moveWindow(s.controls[idActiveProject], xCenter, top, center, 26)
	moveWindow(s.controls[idActivePath], xCenter, top+28, center, 20)
	moveWindow(s.controls[idSummary], xCenter, top+52, center, 22)
	moveWindow(s.controls[idIntentLabel], xCenter, top+82, center, 20)
	moveWindow(s.controls[idIntent], xCenter, top+106, center, 92)
	moveWindow(s.controls[idDelegate], xCenter, top+206, 205, 34)
	moveWindow(s.controls[idTasksLabel], xCenter, top+250, center, 20)
	moveWindow(s.controls[idTaskList], xCenter, top+274, center, 132)
	moveWindow(s.controls[idTaskStatus], xCenter, top+414, center, 22)
	moveWindow(s.controls[idNextAction], xCenter, top+440, center, 42)
	moveWindow(s.controls[idReportLabel], xCenter, top+490, center, 20)
	reportHeight := height - 656
	if reportHeight < 120 {
		reportHeight = 120
	}
	moveWindow(s.controls[idReport], xCenter, top+514, center, reportHeight)
	buttonY := top + 522 + reportHeight
	moveWindow(s.controls[idCancelTask], xCenter, buttonY, 92, 30)
	moveWindow(s.controls[idOpenReview], xCenter+100, buttonY, 112, 30)
	moveWindow(s.controls[idRetryReview], xCenter+220, buttonY, 112, 30)
	moveWindow(s.controls[idCopyBranch], xCenter+340, buttonY, 140, 30)
	moveWindow(s.controls[idAttentionLabel], xCenter, buttonY+40, center, 18)
	moveWindow(s.controls[idAnswer], xCenter, buttonY+62, center-150, 30)
	moveWindow(s.controls[idResumeTask], xCenter+center-142, buttonY+62, 142, 30)

	moveWindow(s.controls[idNotesLabel], xRight, top, right, 20)
	moveWindow(s.controls[idNotes], xRight, top+24, right, height-98)
	moveWindow(s.controls[idSaveNotes], xRight, top+height-66, right, 32)
}

func (s *Shell) layoutSettings(x, y, width, height int) {
	gap := 18
	left := (width - gap) / 2
	right := width - left - gap
	xRight := x + left + gap
	top := y + 8
	moveWindow(s.controls[idSettingsTitle], x, top, width, 24)

	moveWindow(s.controls[idProvidersLabel], x, top+38, left, 20)
	moveWindow(s.controls[idProviderList], x, top+62, left, 190)
	moveWindow(s.controls[idConnectProvider], x, top+260, 150, 32)
	moveWindow(s.controls[idRescanProviders], x+158, top+260, 100, 32)
	moveWindow(s.controls[idProtectWork], x, top+302, left, 26)
	moveWindow(s.controls[idAllowMetered], x, top+332, left, 26)
	moveWindow(s.controls[idMCPLabel], x, top+370, left, 20)
	moveWindow(s.controls[idMCPStatus], x, top+394, left, 48)
	moveWindow(s.controls[idCopyMCP], x, top+450, 180, 32)
	moveWindow(s.controls[idRunnerLabel], x, top+494, left, 20)
	moveWindow(s.controls[idRunnerHost], x, top+518, left, 28)
	moveWindow(s.controls[idHarnessLabel], x, top+554, left, 20)
	moveWindow(s.controls[idHarnessCommand], x, top+578, left, 28)
	moveWindow(s.controls[idNotifyLabel], x, top+614, left, 20)
	moveWindow(s.controls[idNotifyCommand], x, top+638, left, 28)
	moveWindow(s.controls[idSaveRouting], x, top+676, 180, 32)

	moveWindow(s.controls[idReviewLabel], xRight, top+38, right, 20)
	moveWindow(s.controls[idPublishReviews], xRight, top+64, right, 26)
	moveWindow(s.controls[idReviewRemote], xRight, top+96, right, 28)
	moveWindow(s.controls[idSaveReviewPolicy], xRight, top+132, 170, 32)
	moveWindow(s.controls[idVaultLabel], xRight, top+184, right, 20)
	moveWindow(s.controls[idSecretName], xRight, top+210, right, 28)
	moveWindow(s.controls[idSecretValue], xRight, top+246, right, 28)
	moveWindow(s.controls[idSaveSecret], xRight, top+282, 150, 32)
	moveWindow(s.controls[idSecretList], xRight, top+322, right, 172)
	moveWindow(s.controls[idMaintenanceLabel], xRight, top+518, right, 20)
	moveWindow(s.controls[idRunUpdater], xRight, top+544, 230, 34)
}

func (s *Shell) refresh() {
	snapshot := BuildSnapshot(s.eng, s.selectedTaskID)
	s.selectedTaskID = snapshot.SelectedTaskID
	s.refreshGlobalStatus()
	s.refreshProjects(snapshot)
	s.refreshTasks(snapshot)
	if s.page == pageSettings {
		s.refreshSettings(snapshot)
	}
}

func (s *Shell) refreshGlobalStatus() {
	st := s.eng.State()
	summary := core.SummarizeTasks(st.Tasks)
	text := fmt.Sprintf("%d working  ·  %d need you  ·  %d completed  ·  %d failed", summary.Active, summary.NeedsHuman, summary.Completed, summary.Failed)
	if s.mcpURL != "" {
		text += "  ·  Chat bridge online"
	}
	setWindowText(s.controls[idGlobalStatus], text)
}

func (s *Shell) refreshProjects(snapshot Snapshot) {
	selectedProjectID := snapshot.ActiveProjectID
	procSendMessageW.Call(s.controls[idProjectList], lbResetContent, 0, 0)
	s.projectIDs = nil
	selection := -1
	for i, project := range snapshot.Projects {
		mark := "  "
		if project.Pinned {
			mark = "★ "
		}
		line := fmt.Sprintf("%s%s  ·  %d active  ·  %d need you", mark, project.Name, project.Summary.Active, project.Summary.NeedsHuman)
		ptr := wstr(line)
		procSendMessageW.Call(s.controls[idProjectList], lbAddString, 0, uintptr(unsafe.Pointer(ptr)))
		s.projectIDs = append(s.projectIDs, project.ID)
		if project.ID == selectedProjectID {
			selection = i
		}
	}
	if selection >= 0 {
		procSendMessageW.Call(s.controls[idProjectList], lbSetCurSel, uintptr(selection), 0)
	}
	if snapshot.ActiveProjectID == "" {
		setWindowText(s.controls[idActiveProject], "No project selected")
		setWindowText(s.controls[idActivePath], "Add a repository to start delegating work.")
		setWindowText(s.controls[idSummary], "")
		if s.editorProjectID != "" {
			s.editorProjectID = ""
			setWindowText(s.controls[idProjectName], "")
			setWindowText(s.controls[idNotes], "")
		}
		return
	}
	setWindowText(s.controls[idActiveProject], snapshot.ActiveName)
	setWindowText(s.controls[idActivePath], snapshot.ActivePath)
	setWindowText(s.controls[idSummary], fmt.Sprintf("%d working · %d need you · %d ready · %d failed", snapshot.Summary.Active, snapshot.Summary.NeedsHuman, snapshot.Summary.Completed, snapshot.Summary.Failed))
	if s.editorProjectID != snapshot.ActiveProjectID {
		s.editorProjectID = snapshot.ActiveProjectID
		setWindowText(s.controls[idProjectName], snapshot.ActiveName)
		setWindowText(s.controls[idNotes], snapshot.ActiveNotes)
	}
}

func (s *Shell) refreshTasks(snapshot Snapshot) {
	procSendMessageW.Call(s.controls[idTaskList], lbResetContent, 0, 0)
	s.taskIDs = nil
	selection := -1
	for i, task := range snapshot.Tasks {
		line := fmt.Sprintf("[%s] %s  ·  %s", strings.ToUpper(task.StatusLabel), task.Title, task.ProviderLabel)
		ptr := wstr(line)
		procSendMessageW.Call(s.controls[idTaskList], lbAddString, 0, uintptr(unsafe.Pointer(ptr)))
		s.taskIDs = append(s.taskIDs, task.ID)
		if task.ID == snapshot.SelectedTaskID {
			selection = i
		}
	}
	if selection >= 0 {
		procSendMessageW.Call(s.controls[idTaskList], lbSetCurSel, uintptr(selection), 0)
	}
	s.showSelectedTask(snapshot)
}

func (s *Shell) showSelectedTask(snapshot Snapshot) {
	item, ok := snapshot.SelectedTask()
	if !ok {
		setWindowText(s.controls[idTaskStatus], "No task selected")
		setWindowText(s.controls[idNextAction], "Describe an outcome above and Workbench will run it without supervision.")
		setWindowText(s.controls[idReport], "")
		procEnableWindow.Call(s.controls[idCancelTask], 0)
		procEnableWindow.Call(s.controls[idResumeTask], 0)
		procEnableWindow.Call(s.controls[idAnswer], 0)
		procEnableWindow.Call(s.controls[idOpenReview], 0)
		procEnableWindow.Call(s.controls[idRetryReview], 0)
		procEnableWindow.Call(s.controls[idCopyBranch], 0)
		return
	}
	setWindowText(s.controls[idTaskStatus], item.StatusLabel+"  ·  "+item.ProviderLabel)
	setWindowText(s.controls[idNextAction], item.NextAction)
	var report strings.Builder
	if task, exists := s.eng.Task(item.ID); exists {
		if strings.TrimSpace(task.Output) != "" {
			report.WriteString(strings.TrimSpace(task.Output))
		}
		if strings.TrimSpace(task.Error) != "" {
			if report.Len() > 0 {
				report.WriteString("\r\n\r\n")
			}
			report.WriteString("ERROR\r\n" + strings.TrimSpace(task.Error))
		}
		if len(task.Attempts) > 0 {
			if report.Len() > 0 {
				report.WriteString("\r\n\r\n")
			}
			report.WriteString("Activity\r\n- " + strings.Join(task.Attempts, "\r\n- "))
		}
	}
	setWindowText(s.controls[idReport], report.String())
	active := item.Status == core.TaskQueued || item.Status == core.TaskRouting || item.Status == core.TaskRunning
	procEnableWindow.Call(s.controls[idCancelTask], boolWord(active))
	procEnableWindow.Call(s.controls[idResumeTask], boolWord(item.NeedsHuman))
	procEnableWindow.Call(s.controls[idAnswer], boolWord(item.NeedsHuman))
	if !item.NeedsHuman {
		setWindowText(s.controls[idAnswer], "")
	}
	openReview := item.PullRequestStatus == core.ReviewPullRequestAvailable && item.PullRequestNumber > 0
	procEnableWindow.Call(s.controls[idOpenReview], boolWord(openReview))
	retryReview := item.Terminal && item.ReviewBranch != "" && (item.PublicationStatus == core.ReviewPublicationFailed || item.PullRequestStatus == core.ReviewPullRequestUnavailable)
	procEnableWindow.Call(s.controls[idRetryReview], boolWord(retryReview))
	procEnableWindow.Call(s.controls[idCopyBranch], boolWord(item.ReviewBranch != ""))
}

func boolWord(value bool) uintptr {
	if value {
		return 1
	}
	return 0
}
