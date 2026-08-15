//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
	"github.com/DaisyCloverSoftware/workbench/internal/mcp"
	"github.com/DaisyCloverSoftware/workbench/internal/platform"
)

const appVersion = "0.7.0"

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_BORDER           = 0x00800000
	WS_VSCROLL          = 0x00200000
	WS_TABSTOP          = 0x00010000
	WS_EX_CLIENTEDGE    = 0x00000200
	CW_USEDEFAULT       = ^uint32(0x7fffffff)
	SW_SHOW             = 5

	ES_MULTILINE    = 0x0004
	ES_AUTOVSCROLL  = 0x0040
	ES_AUTOHSCROLL  = 0x0080
	ES_READONLY     = 0x0800
	ES_PASSWORD     = 0x0020
	LBS_NOTIFY      = 0x0001
	BS_PUSHBUTTON   = 0x00000000
	BS_AUTOCHECKBOX = 0x00000003

	WM_CREATE          = 0x0001
	WM_DESTROY         = 0x0002
	WM_SIZE            = 0x0005
	WM_COMMAND         = 0x0111
	WM_CLOSE           = 0x0010
	WM_SETFONT         = 0x0030
	WM_CTLCOLORBTN     = 0x0135
	WM_CTLCOLOREDIT    = 0x0133
	WM_CTLCOLORLISTBOX = 0x0134
	WM_CTLCOLORSTATIC  = 0x0138
	WM_APP_REFRESH     = 0x8001

	LB_ADDSTRING    = 0x0180
	LB_RESETCONTENT = 0x0184
	LB_GETCURSEL    = 0x0188
	LB_SETCURSEL    = 0x0186
	LBN_SELCHANGE   = 1
	EM_SETCUEBANNER = 0x1501
	BM_GETCHECK     = 0x00F0
	BM_SETCHECK     = 0x00F1
	BST_CHECKED     = 1

	MB_OK              = 0x00000000
	MB_ICONERROR       = 0x00000010
	MB_ICONINFORMATION = 0x00000040
	MB_ICONWARNING     = 0x00000030
	MB_YESNO           = 0x00000004
	IDYES              = 6

	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002
	GMEM_ZEROINIT  = 0x0040

	COLOR_WINDOW     = 5
	DEFAULT_GUI_FONT = 17
)

const (
	idProviderList      = 1001
	idConnect           = 1002
	idRescan            = 1003
	idCopyMCP           = 1004
	idProject           = 1101
	idBrowse            = 1102
	idIntent            = 1103
	idDelegate          = 1104
	idCancel            = 1105
	idTaskList          = 1106
	idOutput            = 1107
	idAnswer            = 1108
	idResume            = 1109
	idNotes             = 1201
	idSaveNotes         = 1202
	idSecretName        = 1203
	idSecretValue       = 1204
	idSaveSecret        = 1205
	idSecretList        = 1206
	idProtectWork       = 1207
	idAllowMetered      = 1208
	idHarnessCmd        = 1209
	idNotifyCmd         = 1210
	idSaveSettings      = 1211
	idHarnessHost       = 1212
	idPublishReviews    = 1213
	idReviewRemote      = 1214
	idSaveReviewPolicy  = 1215
	idTitle             = 1301
	idLabelRouter       = 1302
	idLabelProject      = 1303
	idLabelIntent       = 1304
	idLabelTasks        = 1305
	idLabelReport       = 1306
	idLabelAttention    = 1307
	idLabelKnowledge    = 1308
	idLabelNotes        = 1309
	idLabelVault        = 1310
	idLabelHarness      = 1311
	idLabelNotify       = 1312
	idMCPStatus         = 1313
	idLabelHarnessHost  = 1314
	idLabelReviewPolicy = 1315
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procRegisterClassExW      = user32.NewProc("RegisterClassExW")
	procCreateWindowExW       = user32.NewProc("CreateWindowExW")
	procDefWindowProcW        = user32.NewProc("DefWindowProcW")
	procShowWindow            = user32.NewProc("ShowWindow")
	procUpdateWindow          = user32.NewProc("UpdateWindow")
	procGetMessageW           = user32.NewProc("GetMessageW")
	procTranslateMessage      = user32.NewProc("TranslateMessage")
	procDispatchMessageW      = user32.NewProc("DispatchMessageW")
	procPostQuitMessage       = user32.NewProc("PostQuitMessage")
	procPostMessageW          = user32.NewProc("PostMessageW")
	procSendMessageW          = user32.NewProc("SendMessageW")
	procMoveWindow            = user32.NewProc("MoveWindow")
	procGetClientRect         = user32.NewProc("GetClientRect")
	procGetWindowTextLengthW  = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW        = user32.NewProc("GetWindowTextW")
	procSetWindowTextW        = user32.NewProc("SetWindowTextW")
	procMessageBoxW           = user32.NewProc("MessageBoxW")
	procEnableWindow          = user32.NewProc("EnableWindow")
	procGetStockObject        = gdi32.NewProc("GetStockObject")
	procCreateSolidBrush      = gdi32.NewProc("CreateSolidBrush")
	procSetBkColor            = gdi32.NewProc("SetBkColor")
	procSetTextColor          = gdi32.NewProc("SetTextColor")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	procOpenClipboard         = user32.NewProc("OpenClipboard")
	procCloseClipboard        = user32.NewProc("CloseClipboard")
	procEmptyClipboard        = user32.NewProc("EmptyClipboard")
	procSetClipboardData      = user32.NewProc("SetClipboardData")
	procGlobalAlloc           = kernel32.NewProc("GlobalAlloc")
	procGlobalLock            = kernel32.NewProc("GlobalLock")
	procGlobalUnlock          = kernel32.NewProc("GlobalUnlock")
	procSHBrowseForFolderW    = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW  = shell32.NewProc("SHGetPathFromIDListW")
	procCoTaskMemFree         = ole32.NewProc("CoTaskMemFree")
)

type point struct{ X, Y int32 }
type msg struct {
	Hwnd           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             point
	LPrivate       uint32
}
type rect struct{ Left, Top, Right, Bottom int32 }
type wndClassEx struct {
	Size                                                            uint32
	Style                                                           uint32
	WndProc                                                         uintptr
	ClsExtra, WndExtra                                              int32
	Instance, Icon, Cursor, Background, MenuName, ClassName, IconSm uintptr
}
type browseInfo struct {
	Owner, Root        uintptr
	DisplayName, Title *uint16
	Flags              uint32
	Callback, LParam   uintptr
	Image              int32
}

type app struct {
	hwnd          uintptr
	eng           *core.Engine
	mcp           *mcp.Server
	mcpURL        string
	mcpErr        string
	font          uintptr
	brush         uintptr
	controls      map[int]uintptr
	providerIDs   []string
	taskIDs       []string
	policyProject string
}

var g *app

func main() {
	store, err := core.NewStore()
	if err != nil {
		fatalNative("Workbench could not open its local state", err)
		return
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		fatalNative("Workbench could not initialise", err)
		return
	}
	st := eng.State()
	var srv *mcp.Server
	var mcpURL, mcpErr string
	for port := st.Preferences.MCPPort; port < st.Preferences.MCPPort+20; port++ {
		s := mcp.New(eng, port, st.Preferences.MCPToken)
		if err := s.Start(); err == nil {
			srv = s
			mcpURL = s.URL()
			if port != st.Preferences.MCPPort {
				p := st.Preferences
				p.MCPPort = port
				_ = eng.SavePreferences(p)
			}
			break
		} else {
			mcpErr = err.Error()
		}
	}
	g = &app{eng: eng, mcp: srv, mcpURL: mcpURL, mcpErr: mcpErr, controls: map[int]uintptr{}}
	if err := g.run(); err != nil {
		fatalNative("Workbench UI failed", err)
	}
}

func (a *app) run() error {
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	className := wstr("WorkbenchNativeWindow")
	icon, _, _ := user32.NewProc("LoadIconW").Call(0, 32512)
	cursor, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
	a.brush, _, _ = procCreateSolidBrush.Call(uintptr(rgb(24, 27, 32)))
	wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: syscall.NewCallback(wndProc), Instance: instance, Icon: icon, Cursor: cursor, Background: a.brush, ClassName: uintptr(unsafe.Pointer(className)), IconSm: icon}
	if r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return e
	}
	title := wstr("Workbench " + appVersion + " — AI Coder Control Plane")
	hwnd, _, e := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), WS_OVERLAPPEDWINDOW|WS_VISIBLE, 100, 70, 1500, 930, 0, 0, instance, 0)
	if hwnd == 0 {
		return e
	}
	a.hwnd = hwnd
	useDark := uint32(1)
	_, _, _ = procDwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&useDark)), unsafe.Sizeof(useDark))
	a.eng.Subscribe(func() {
		if a.hwnd != 0 {
			procPostMessageW.Call(a.hwnd, WM_APP_REFRESH, 0, 0)
		}
	})
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	if a.mcp != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = a.mcp.Close(ctx)
		cancel()
	}
	return nil
}

func wndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if g == nil {
		return def(hwnd, message, wParam, lParam)
	}
	switch message {
	case WM_CREATE:
		g.hwnd = hwnd
		g.createControls()
		g.refreshAll()
		g.layout()
		return 0
	case WM_SIZE:
		g.layout()
		return 0
	case WM_APP_REFRESH:
		g.refreshAll()
		return 0
	case WM_COMMAND:
		id := int(uint16(wParam & 0xffff))
		notify := uint16((wParam >> 16) & 0xffff)
		g.command(id, notify)
		return 0
	case WM_CTLCOLORSTATIC, WM_CTLCOLORBTN, WM_CTLCOLOREDIT, WM_CTLCOLORLISTBOX:
		hdc := wParam
		procSetTextColor.Call(hdc, uintptr(rgb(232, 235, 240)))
		procSetBkColor.Call(hdc, uintptr(rgb(24, 27, 32)))
		return g.brush
	case WM_CLOSE:
		user32.NewProc("DestroyWindow").Call(hwnd)
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	return def(hwnd, message, wParam, lParam)
}
func def(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), w, l)
	return r
}

func (a *app) createControls() {
	a.font, _, _ = procGetStockObject.Call(DEFAULT_GUI_FONT)
	a.static("WORKBENCH  ·  AI CODER CONTROL PLANE  ·  USE CHAT FOR BRAINS, THE ROUTER FOR HANDS", idTitle)
	a.static("AI ROUTER", idLabelRouter)
	a.static("Project / repository", idLabelProject)
	a.static("What outcome do you want?", idLabelIntent)
	a.static("Task queue", idLabelTasks)
	a.static("Worker report", idLabelReport)
	a.static("Only when Workbench genuinely needs you", idLabelAttention)
	a.static("KNOWLEDGE + VAULT", idLabelKnowledge)
	a.static("Scratchpad / parked notes", idLabelNotes)
	a.static("Encrypted vault · Windows DPAPI · values never shown to AI", idLabelVault)
	a.static("OpenClaw SSH host · optional", idLabelHarnessHost)
	a.static("Custom harness command template · optional", idLabelHarness)
	a.static("Human interrupt command · optional · {message}", idLabelNotify)
	a.static("Review publication · explicit policy · kept outside AI/task state", idLabelReviewPolicy)
	a.static("", idMCPStatus)
	a.ctrl(idProviderList, "LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOTIFY)
	a.ctrl(idConnect, "BUTTON", "Connect selected", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idRescan, "BUTTON", "Rescan", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idCopyMCP, "BUTTON", "Copy ChatGPT MCP setup", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idProject, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL)
	a.ctrl(idBrowse, "BUTTON", "Browse…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idIntent, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_MULTILINE|ES_AUTOVSCROLL|WS_VSCROLL)
	a.ctrl(idDelegate, "BUTTON", "Delegate autonomously", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idCancel, "BUTTON", "Cancel task", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idTaskList, "LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOTIFY)
	a.ctrl(idOutput, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|ES_MULTILINE|ES_AUTOVSCROLL|WS_VSCROLL|ES_READONLY)
	a.ctrl(idAnswer, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL)
	a.ctrl(idResume, "BUTTON", "Answer + resume", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idNotes, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_MULTILINE|ES_AUTOVSCROLL|WS_VSCROLL)
	a.ctrl(idSaveNotes, "BUTTON", "Save notes", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idSecretName, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL)
	a.ctrl(idSecretValue, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL|ES_PASSWORD)
	a.ctrl(idSaveSecret, "BUTTON", "Encrypt + store", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idSecretList, "LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL)
	a.ctrl(idProtectWork, "BUTTON", "Protect scarce Work/Codex usage", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX)
	a.ctrl(idAllowMetered, "BUTTON", "Allow metered API fallback", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX)
	a.ctrl(idHarnessHost, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL)
	a.ctrl(idHarnessCmd, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL)
	a.ctrl(idNotifyCmd, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL)
	a.ctrl(idSaveSettings, "BUTTON", "Save routing settings", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idPublishReviews, "BUTTON", "Publish prepared review branches", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX)
	a.ctrl(idReviewRemote, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL)
	a.ctrl(idSaveReviewPolicy, "BUTTON", "Save review policy", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	cue(a.controls[idProject], "Choose a local repository folder")
	cue(a.controls[idIntent], "Describe the outcome. Workbench chooses the cheapest eligible worker and keeps going.")
	cue(a.controls[idAnswer], "Only type here when the task above says NEEDS YOU")
	cue(a.controls[idSecretName], "secret name, e.g. prod/ssh")
	cue(a.controls[idSecretValue], "secret value")
	cue(a.controls[idHarnessHost], "e.g. user@your-agent-host.tailnet.ts.net")
	cue(a.controls[idHarnessCmd], "optional advanced adapter: command {project} {prompt}")
	cue(a.controls[idNotifyCmd], "optional: OpenClaw / WhatsApp command using {message}")
	cue(a.controls[idReviewRemote], "explicit review remote: HTTPS, SSH, scp-style SSH, or local path · no embedded credentials")
}

func (a *app) static(text string, id int) uintptr {
	return a.ctrl(id, "STATIC", text, WS_CHILD|WS_VISIBLE)
}
func (a *app) ctrl(id int, class, text string, style uintptr) uintptr {
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	c, t := wstr(class), wstr(text)
	exStyle := uintptr(0)
	if class == "EDIT" || class == "LISTBOX" {
		exStyle = WS_EX_CLIENTEDGE
	}
	h, _, _ := procCreateWindowExW.Call(exStyle, uintptr(unsafe.Pointer(c)), uintptr(unsafe.Pointer(t)), style, 0, 0, 10, 10, a.hwnd, uintptr(id), instance, 0)
	if id != 0 {
		a.controls[id] = h
	}
	procSendMessageW.Call(h, WM_SETFONT, a.font, 1)
	return h
}

func (a *app) layout() {
	if a.hwnd == 0 {
		return
	}
	var r rect
	procGetClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&r)))
	w := int(r.Right - r.Left)
	h := int(r.Bottom - r.Top)
	pad := 14
	top := 46
	bottom := 34
	left := 320
	right := 390
	if w < 1250 {
		left = 280
		right = 340
	}
	center := w - left - right - pad*4
	if center < 420 {
		center = 420
	}
	// We locate controls by ID only; static labels are positioned in creation order by finding child windows is cumbersome.
	// The native controls are the interactive surface; section labels are deliberately allowed to flow behind them on small windows.
	xL := pad
	xC := left + pad*2
	xR := left + center + pad*3
	move(a.controls[idTitle], pad, 12, w-pad*2, 22)
	move(a.controls[idLabelRouter], xL, top-22, left, 20)
	move(a.controls[idLabelProject], xC, top-22, center, 20)
	move(a.controls[idLabelKnowledge], xR, top-22, right, 20)
	move(a.controls[idProviderList], xL, top, left, h-top-bottom-165)
	move(a.controls[idConnect], xL, h-bottom-150, left/2-5, 34)
	move(a.controls[idRescan], xL+left/2+5, h-bottom-150, left/2-5, 34)
	move(a.controls[idCopyMCP], xL, h-bottom-108, left, 34)
	// center
	move(a.controls[idProject], xC, top, center-100, 30)
	move(a.controls[idBrowse], xC+center-92, top, 92, 30)
	move(a.controls[idLabelIntent], xC, top+36, center, 18)
	move(a.controls[idIntent], xC, top+56, center, 98)
	move(a.controls[idDelegate], xC, top+164, 210, 36)
	move(a.controls[idCancel], xC+220, top+164, 120, 36)
	move(a.controls[idLabelTasks], xC, top+208, center, 18)
	move(a.controls[idTaskList], xC, top+228, center, 134)
	move(a.controls[idLabelReport], xC, top+370, center, 18)
	move(a.controls[idOutput], xC, top+390, center, h-top-bottom-516)
	move(a.controls[idLabelAttention], xC, h-bottom-136, center, 18)
	move(a.controls[idAnswer], xC, h-bottom-112, center-145, 31)
	move(a.controls[idResume], xC+center-135, h-bottom-112, 135, 31)
	// right
	move(a.controls[idLabelNotes], xR, top-2, right, 18)
	move(a.controls[idNotes], xR, top+20, right, 170)
	move(a.controls[idSaveNotes], xR, top+200, right, 32)
	move(a.controls[idLabelVault], xR, top+236, right, 18)
	move(a.controls[idSecretName], xR, top+250, right, 28)
	move(a.controls[idSecretValue], xR, top+286, right, 28)
	move(a.controls[idSaveSecret], xR, top+322, right, 31)
	move(a.controls[idSecretList], xR, top+360, right, 95)
	move(a.controls[idProtectWork], xR, top+466, right, 26)
	move(a.controls[idAllowMetered], xR, top+494, right, 26)
	move(a.controls[idLabelHarnessHost], xR, top+524, right, 18)
	move(a.controls[idHarnessHost], xR, top+544, right, 28)
	move(a.controls[idLabelHarness], xR, top+578, right, 18)
	move(a.controls[idHarnessCmd], xR, top+598, right, 28)
	move(a.controls[idLabelNotify], xR, top+632, right, 18)
	move(a.controls[idNotifyCmd], xR, top+652, right, 28)
	move(a.controls[idSaveSettings], xR, top+686, right, 32)
	move(a.controls[idLabelReviewPolicy], xR, top+724, right, 18)
	move(a.controls[idPublishReviews], xR, top+744, right, 26)
	move(a.controls[idReviewRemote], xR, top+774, right, 28)
	move(a.controls[idSaveReviewPolicy], xR, top+808, right, 32)
	move(a.controls[idMCPStatus], xL, h-bottom-67, left, 42)
}
func move(h uintptr, x, y, w, hgt int) {
	if h != 0 {
		procMoveWindow.Call(h, uintptr(x), uintptr(y), uintptr(w), uintptr(hgt), 1)
	}
}

func (a *app) refreshAll() {
	st := a.eng.State()
	setText(a.controls[idProject], st.ProjectPath)
	setText(a.controls[idNotes], st.Notes)
	setText(a.controls[idHarnessHost], st.Preferences.OpenClawSSHHost)
	setText(a.controls[idHarnessCmd], st.Preferences.OpenClawCommand)
	setText(a.controls[idNotifyCmd], st.Preferences.NotificationCommand)
	if st.Preferences.AvoidWorkUsage {
		procSendMessageW.Call(a.controls[idProtectWork], BM_SETCHECK, BST_CHECKED, 0)
	} else {
		procSendMessageW.Call(a.controls[idProtectWork], BM_SETCHECK, 0, 0)
	}
	if st.Preferences.AllowMeteredAPI {
		procSendMessageW.Call(a.controls[idAllowMetered], BM_SETCHECK, BST_CHECKED, 0)
	} else {
		procSendMessageW.Call(a.controls[idAllowMetered], BM_SETCHECK, 0, 0)
	}
	a.refreshPublicationPolicy(st.ProjectPath)
	a.refreshProviders()
	a.refreshTasks()
	a.refreshSecrets()
	status := "MCP: unavailable"
	if a.mcpURL != "" {
		status = "MCP: " + a.mcpURL + "  ·  bearer auth on  ·  Work protection on"
	}
	setText(a.controls[idMCPStatus], status)
}
func (a *app) refreshPublicationPolicy(project string) {
	project = strings.TrimSpace(project)
	if project == a.policyProject {
		return
	}
	a.policyProject = project
	procSendMessageW.Call(a.controls[idPublishReviews], BM_SETCHECK, 0, 0)
	setText(a.controls[idReviewRemote], "")
	if project == "" {
		return
	}
	policy, configured, err := core.PublicationPolicyFor(project)
	if err != nil || !configured {
		return
	}
	if policy.Mode == core.PublicationPublish {
		procSendMessageW.Call(a.controls[idPublishReviews], BM_SETCHECK, BST_CHECKED, 0)
		setText(a.controls[idReviewRemote], policy.RemoteURL)
	}
}

func (a *app) refreshProviders() {
	ps := a.eng.Providers()
	procSendMessageW.Call(a.controls[idProviderList], LB_RESETCONTENT, 0, 0)
	a.providerIDs = nil
	st := a.eng.State()
	for _, p := range ps {
		mark := "○"
		status := p.Status
		if p.Installed {
			mark = "●"
		}
		if p.ID == "openclaw" && strings.TrimSpace(st.Preferences.OpenClawSSHHost) != "" {
			mark = "●"
			status = "remote configured · " + st.Preferences.OpenClawSSHHost
		}
		line := fmt.Sprintf("%s %s  |  %s  |  %s", mark, p.Name, status, p.Cost)
		lp := wstr(line)
		procSendMessageW.Call(a.controls[idProviderList], LB_ADDSTRING, 0, uintptr(unsafe.Pointer(lp)))
		a.providerIDs = append(a.providerIDs, p.ID)
	}
}
func (a *app) refreshTasks() {
	selectedID := a.selectedTaskID()
	st := a.eng.State()
	procSendMessageW.Call(a.controls[idTaskList], LB_RESETCONTENT, 0, 0)
	a.taskIDs = nil
	sel := -1
	for i, t := range st.Tasks {
		provider := t.ProviderID
		if provider == "" {
			provider = "router"
		}
		line := fmt.Sprintf("[%s] %s  ·  %s", strings.ToUpper(string(t.Status)), t.Title, provider)
		lp := wstr(line)
		procSendMessageW.Call(a.controls[idTaskList], LB_ADDSTRING, 0, uintptr(unsafe.Pointer(lp)))
		a.taskIDs = append(a.taskIDs, t.ID)
		if t.ID == selectedID {
			sel = i
		}
	}
	if len(st.Tasks) > 0 {
		if sel < 0 {
			sel = 0
		}
		procSendMessageW.Call(a.controls[idTaskList], LB_SETCURSEL, uintptr(sel), 0)
		a.showTask(sel)
	} else {
		setText(a.controls[idOutput], "No tasks yet. Describe an outcome above and delegate it. Workbench will route zero-marginal/included workers first and protect scarce Work/Codex usage.")
	}
}
func (a *app) refreshSecrets() {
	st := a.eng.State()
	procSendMessageW.Call(a.controls[idSecretList], LB_RESETCONTENT, 0, 0)
	for _, s := range st.Secrets {
		lp := wstr("vault://" + s.Name)
		procSendMessageW.Call(a.controls[idSecretList], LB_ADDSTRING, 0, uintptr(unsafe.Pointer(lp)))
	}
}

func (a *app) command(id int, notify uint16) {
	switch id {
	case idProviderList:
		if notify == LBN_SELCHANGE {
			a.showProvider()
		}
	case idTaskList:
		if notify == LBN_SELCHANGE {
			idx := listSel(a.controls[idTaskList])
			a.showTask(idx)
		}
	case idConnect:
		a.connectSelected()
	case idRescan:
		go a.eng.RescanProviders()
	case idCopyMCP:
		a.copyMCP()
	case idBrowse:
		if p := browseFolder(a.hwnd); p != "" {
			setText(a.controls[idProject], p)
		}
	case idDelegate:
		intent := getText(a.controls[idIntent])
		project := getText(a.controls[idProject])
		if _, err := a.eng.Delegate("desktop", intent, project); err != nil {
			msgbox(a.hwnd, "Cannot delegate", err.Error(), MB_ICONWARNING)
		} else {
			setText(a.controls[idIntent], "")
		}
	case idCancel:
		if t := a.selectedTaskID(); t != "" {
			_ = a.eng.Cancel(t)
		}
	case idSaveNotes:
		a.saveNotes()
	case idSaveSecret:
		a.saveSecret()
	case idResume:
		id := a.selectedTaskID()
		answer := getText(a.controls[idAnswer])
		if err := a.eng.ResolveAttention(id, answer); err != nil {
			msgbox(a.hwnd, "Cannot resume", err.Error(), MB_ICONWARNING)
		} else {
			setText(a.controls[idAnswer], "")
		}
	case idSaveSettings:
		a.saveSettings()
	case idSaveReviewPolicy:
		a.saveReviewPolicy()
	}
}
func (a *app) showProvider() {
	idx := listSel(a.controls[idProviderList])
	if idx < 0 || idx >= len(a.providerIDs) {
		return
	}
	id := a.providerIDs[idx]
	for _, p := range a.eng.Providers() {
		if p.ID == id {
			msgbox(a.hwnd, p.Name, p.Capability+"\n\n"+p.Status+"\n\n"+p.Notes, MB_ICONINFORMATION)
			return
		}
	}
}
func (a *app) connectSelected() {
	idx := listSel(a.controls[idProviderList])
	if idx < 0 || idx >= len(a.providerIDs) {
		msgbox(a.hwnd, "AI Router", "Select a provider first.", MB_ICONINFORMATION)
		return
	}
	id := a.providerIDs[idx]
	if id == "openclaw" {
		host := a.eng.State().Preferences.OpenClawSSHHost
		if strings.TrimSpace(host) == "" {
			msgbox(a.hwnd, "OpenClaw", "Enter the OpenClaw SSH host under routing settings, save it, then Connect selected will test the real remote CLI.", MB_ICONINFORMATION)
			return
		}
		out, err := core.TestOpenClawSSH(host)
		if err != nil {
			msgbox(a.hwnd, "OpenClaw connection failed", err.Error()+"\n\n"+out, MB_ICONWARNING)
			return
		}
		msgbox(a.hwnd, "OpenClaw connected", "Remote OpenClaw responded over SSH.\n\n"+out, MB_ICONINFORMATION)
		return
	}
	if id == "chatgpt" {
		a.copyMCP()
		msgbox(a.hwnd, "ChatGPT bridge", "Workbench's local MCP endpoint and bearer token were copied to the clipboard. Connect that endpoint through a supported private MCP/plugin tunnel. Ordinary Chat can then use Workbench as its hands.", MB_ICONINFORMATION)
		return
	}
	if err := core.StartProviderLogin(id); err != nil {
		msgbox(a.hwnd, "Provider setup", setupHint(id)+"\n\n"+err.Error(), MB_ICONWARNING)
		return
	}
	msgbox(a.hwnd, "Connect provider", "Workbench opened the provider's own sign-in flow. Finish sign-in, then click Rescan. Your provider password is never entered into Workbench.", MB_ICONINFORMATION)
}
func setupHint(id string) string {
	switch id {
	case "codex":
		return "Install the official Codex CLI, then use Connect selected."
	case "claude":
		return "Install Claude Code, then use Connect selected."
	case "copilot":
		return "Install GitHub Copilot CLI, then use Connect selected."
	case "antigravity":
		return "Install Google Antigravity CLI (agy), then use Connect selected."
	case "gemini":
		return "Gemini CLI is now a legacy enterprise/API adapter for this project; individual accounts should use Antigravity CLI."
	case "openclaw":
		return "Set an OpenClaw/harness command template under routing settings, or install its CLI."
	case "grok":
		return "Workbench deliberately does not automate consumer browser sessions. Grok remains a future supported adapter / opt-in API route."
	}
	return "No supported login adapter exists for this provider yet."
}
func (a *app) copyMCP() {
	st := a.eng.State()
	if a.mcpURL == "" {
		msgbox(a.hwnd, "MCP unavailable", "Local MCP server could not start: "+a.mcpErr, MB_ICONERROR)
		return
	}
	text := "Workbench MCP\r\nURL: " + a.mcpURL + "\r\nAuthorization: Bearer " + st.Preferences.MCPToken + "\r\n\r\nPolicy: Use Chat for brains; apply_patch/run_safe_command for hands; delegate_task only for true autonomous exploration; poll get_task without bothering the human for progress."
	if copyClipboard(a.hwnd, text) {
		msgbox(a.hwnd, "Copied", "MCP connection details copied. Treat the bearer token like a local credential.", MB_ICONINFORMATION)
	}
}
func (a *app) saveNotes() {
	notes := getText(a.controls[idNotes])
	if core.LooksSecret(notes) {
		msgbox(a.hwnd, "Secret detected", "Workbench refused to save this scratchpad because it looks like it contains a credential. Put the secret into the encrypted Vault fields instead, then keep only a vault://name reference in your note.", MB_ICONWARNING)
		return
	}
	if err := a.eng.SaveNotes(getText(a.controls[idProject]), notes); err != nil {
		msgbox(a.hwnd, "Save failed", err.Error(), MB_ICONERROR)
	}
}
func (a *app) saveSecret() {
	name := strings.TrimSpace(getText(a.controls[idSecretName]))
	value := getText(a.controls[idSecretValue])
	if name == "" || value == "" {
		msgbox(a.hwnd, "Vault", "Enter both a secret name and value.", MB_ICONINFORMATION)
		return
	}
	cipher, err := platform.ProtectString(value)
	if err != nil {
		msgbox(a.hwnd, "Vault encryption failed", err.Error(), MB_ICONERROR)
		return
	}
	if err := a.eng.AddSecret(core.SecretRef{Name: name, Ciphertext: cipher, CreatedAt: time.Now()}); err != nil {
		msgbox(a.hwnd, "Vault save failed", err.Error(), MB_ICONERROR)
		return
	}
	setText(a.controls[idSecretValue], "")
	setText(a.controls[idSecretName], "")
	msgbox(a.hwnd, "Stored", "Encrypted as vault://"+name+" using Windows DPAPI for this Windows user. The raw value is not exposed to AI tools.", MB_ICONINFORMATION)
}
func (a *app) saveSettings() {
	st := a.eng.State()
	p := st.Preferences
	p.AvoidWorkUsage = buttonChecked(a.controls[idProtectWork])
	p.AllowMeteredAPI = buttonChecked(a.controls[idAllowMetered])
	p.OpenClawSSHHost = getText(a.controls[idHarnessHost])
	p.OpenClawCommand = getText(a.controls[idHarnessCmd])
	p.NotificationCommand = getText(a.controls[idNotifyCmd])
	if err := a.eng.SavePreferences(p); err != nil {
		msgbox(a.hwnd, "Settings failed", err.Error(), MB_ICONERROR)
		return
	}
	msgbox(a.hwnd, "Routing saved", "Router policy updated. Workbench will continue without asking unless a genuine attention boundary is reached.", MB_ICONINFORMATION)
}
func (a *app) saveReviewPolicy() {
	project := strings.TrimSpace(getText(a.controls[idProject]))
	if project == "" {
		msgbox(a.hwnd, "Review policy", "Choose a project repository first.", MB_ICONINFORMATION)
		return
	}

	mode := core.PublicationPrepare
	remote := ""
	if buttonChecked(a.controls[idPublishReviews]) {
		mode = core.PublicationPublish
		remote = strings.TrimSpace(getText(a.controls[idReviewRemote]))
		if remote == "" {
			msgbox(a.hwnd, "Review policy", "Enter the explicit review remote before enabling automatic review-branch publication.", MB_ICONINFORMATION)
			return
		}
	}

	st := a.eng.State()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := core.SavePublicationPolicyForExecutionHosts(ctx, project, mode, remote, st.Preferences.OpenClawSSHHost)
	if err != nil {
		if strings.TrimSpace(result.Local.Project) != "" {
			msgbox(a.hwnd, "Runner policy sync failed", err.Error()+"\n\nThe local review policy was saved. The configured runner was not changed.", MB_ICONWARNING)
			return
		}
		msgbox(a.hwnd, "Review policy failed", err.Error(), MB_ICONERROR)
		return
	}

	a.policyProject = project
	if mode == core.PublicationPrepare {
		setText(a.controls[idReviewRemote], "")
	}
	scope := "Saved for local Workbench execution."
	if result.Runner != nil {
		scope = "Saved locally and synchronised to the configured Workbench runner."
	}
	detail := "Successful coding tasks will create Workbench-owned local review branches. Coding workers still cannot push or publish."
	if mode == core.PublicationPublish {
		detail = "Successful coding tasks may publish only their Workbench-owned review branch to the explicit target. Coding workers still cannot push or choose the target."
	}
	msgbox(a.hwnd, "Review policy saved", scope+"\n\n"+detail, MB_ICONINFORMATION)
}

func (a *app) showTask(idx int) {
	st := a.eng.State()
	if idx < 0 || idx >= len(st.Tasks) {
		return
	}
	t := st.Tasks[idx]
	var b strings.Builder
	fmt.Fprintf(&b, "%s\r\n\r\nStatus: %s\r\nProvider: %s\r\nRoute: %s\r\nConsumes scarce Work: %t\r\n\r\n", t.Title, t.Status, t.ProviderID, t.RouteReason, t.ConsumesWork)
	if t.AttentionQuestion != "" {
		fmt.Fprintf(&b, "NEEDS YOU: %s\r\n\r\n", t.AttentionQuestion)
	}
	if t.Output != "" {
		b.WriteString(t.Output)
		b.WriteString("\r\n\r\n")
	}
	if t.Error != "" {
		b.WriteString("ERROR:\r\n" + t.Error + "\r\n\r\n")
	}
	if len(t.Attempts) > 0 {
		b.WriteString("Routing attempts:\r\n- " + strings.Join(t.Attempts, "\r\n- "))
	}
	setText(a.controls[idOutput], b.String())
	procEnableWindow.Call(a.controls[idResume], boolPtr(t.Status == core.TaskNeedsAttention))
	procEnableWindow.Call(a.controls[idAnswer], boolPtr(t.Status == core.TaskNeedsAttention))
}
func (a *app) selectedTaskID() string {
	idx := listSel(a.controls[idTaskList])
	if idx >= 0 && idx < len(a.taskIDs) {
		return a.taskIDs[idx]
	}
	return ""
}

func wstr(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func getText(h uintptr) string {
	n, _, _ := procGetWindowTextLengthW.Call(h)
	buf := make([]uint16, n+1)
	procGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}
func setText(h uintptr, s string) {
	if h == 0 {
		return
	}
	p := wstr(s)
	procSetWindowTextW.Call(h, uintptr(unsafe.Pointer(p)))
}
func cue(h uintptr, s string) {
	if h == 0 {
		return
	}
	p := wstr(s)
	procSendMessageW.Call(h, EM_SETCUEBANNER, 1, uintptr(unsafe.Pointer(p)))
}
func listSel(h uintptr) int {
	r, _, _ := procSendMessageW.Call(h, LB_GETCURSEL, 0, 0)
	if int32(r) < 0 {
		return -1
	}
	return int(r)
}
func buttonChecked(h uintptr) bool {
	r, _, _ := procSendMessageW.Call(h, BM_GETCHECK, 0, 0)
	return r == BST_CHECKED
}
func boolPtr(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}
func rgb(r, g, b byte) uint32 { return uint32(r) | uint32(g)<<8 | uint32(b)<<16 }
func msgbox(hwnd uintptr, title, text string, flags uintptr) int {
	t, m := wstr(title), wstr(text)
	r, _, _ := procMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), flags|MB_OK)
	return int(r)
}
func fatalNative(title string, err error) { msgbox(0, title, err.Error(), MB_ICONERROR) }

func copyClipboard(hwnd uintptr, text string) bool {
	r, _, _ := procOpenClipboard.Call(hwnd)
	if r == 0 {
		return false
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	u, _ := syscall.UTF16FromString(text)
	size := uintptr(len(u) * 2)
	h, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE|GMEM_ZEROINIT, size)
	if h == 0 {
		return false
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return false
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(u))
	copy(dst, u)
	procGlobalUnlock.Call(h)
	r, _, _ = procSetClipboardData.Call(CF_UNICODETEXT, h)
	return r != 0
}

func browseFolder(owner uintptr) string {
	display := make([]uint16, 260)
	title := wstr("Choose the project/repository Workbench may edit")
	bi := browseInfo{Owner: owner, DisplayName: &display[0], Title: title, Flags: 0x0001 | 0x0040}
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return ""
	}
	defer procCoTaskMemFree.Call(pidl)
	path := make([]uint16, 32768)
	ok, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(path)
}

// Keep imports used in GUI-only builds when Go optimises platform-specific code paths.
var _ = strconv.Itoa
var _ = os.Args
var _ = exec.Command
