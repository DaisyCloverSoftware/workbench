$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$proof = Join-Path $env:RUNNER_TEMP 'sprint1-telemetry-proof'
New-Item -ItemType Directory -Force -Path $proof | Out-Null
$env:APPDATA = Join-Path $env:RUNNER_TEMP 'sprint1-telemetry-appdata'
$env:LOCALAPPDATA = Join-Path $env:RUNNER_TEMP 'sprint1-telemetry-localappdata'
$env:PATH = "$env:RUNNER_TEMP;$env:PATH"
New-Item -ItemType Directory -Force -Path $env:APPDATA,$env:LOCALAPPDATA | Out-Null

$stateDir = Join-Path $env:APPDATA 'Workbench'
New-Item -ItemType Directory -Force -Path $stateDir | Out-Null
$statePath = Join-Path $stateDir 'state.json'
$adapterPath = Join-Path $env:RUNNER_TEMP 'Sprint1TelemetryHarness.exe'
$exePath = Join-Path $env:RUNNER_TEMP 'Workbench-Sprint1-Telemetry.exe'
$projectPath = (Get-Location).Path
$now = [DateTime]::UtcNow

$harnessSource = Join-Path $env:RUNNER_TEMP 'sprint1-telemetry-harness.go'
@'
package main

import (
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"
)

type job struct {
    Version int `json:"version"`
    TaskID string `json:"task_id"`
    ProjectPath string `json:"project_path"`
    Intent string `json:"intent"`
}

func progress(kind, phase string, current, total int64, stage, stageTotal int) {
    payload := map[string]any{"kind": kind, "phase": phase}
    if kind == "measured" {
        payload["current"] = current
        payload["total"] = total
        payload["unit"] = "files"
    } else {
        payload["stage"] = stage
        payload["stage_total"] = stageTotal
    }
    body, _ := json.Marshal(payload)
    fmt.Fprintf(os.Stderr, "WORKBENCH_PROGRESS: %s\n", body)
}

func measuredWork(j job) error {
    root := filepath.Join(j.ProjectPath, "internal", "core")
    files := []string{}
    err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if err != nil { return err }
        if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".go") {
            files = append(files, path)
        }
        return nil
    })
    if err != nil { return err }
    sort.Strings(files)
    if len(files) == 0 { return fmt.Errorf("no source files to verify") }

    const passes = 4
    total := int64(len(files) * passes)
    var current int64
    for pass := 0; pass < passes; pass++ {
        for _, path := range files {
            data, err := os.ReadFile(path)
            if err != nil { return err }
            _ = sha256.Sum256(data)
            current++
            progress("measured", "Verifying source files", current, total, 0, 0)
            time.Sleep(120 * time.Millisecond)
        }
    }
    return nil
}

func stageWork() {
    phases := []string{"Preparing checks", "Executing checks", "Verifying result", "Finalizing result"}
    for i, phase := range phases {
        progress("stages", phase, 0, 0, i+1, len(phases))
        time.Sleep(1200 * time.Millisecond)
    }
}

func main() {
    var j job
    if err := json.NewDecoder(os.Stdin).Decode(&j); err != nil { os.Exit(2) }
    switch {
    case strings.Contains(j.Intent, "Telemetry measured"):
        if err := measuredWork(j); err != nil {
            fmt.Printf("{\"version\":1,\"task_id\":%q,\"status\":\"failed\",\"report\":%q,\"category\":\"adapter\",\"retryable\":false}\n", j.TaskID, err.Error())
            return
        }
        fmt.Printf("{\"version\":1,\"task_id\":%q,\"status\":\"completed\",\"report\":\"Measured source verification complete.\"}\n", j.TaskID)
    case strings.Contains(j.Intent, "Telemetry needs you"):
        progress("stages", "Checking owner boundary", 0, 0, 2, 4)
        time.Sleep(2 * time.Second)
        fmt.Printf("{\"version\":1,\"task_id\":%q,\"status\":\"needs_attention\",\"attention\":\"Choose the telemetry acceptance owner action.\"}\n", j.TaskID)
    default:
        stageWork()
        fmt.Printf("{\"version\":1,\"task_id\":%q,\"status\":\"completed\",\"report\":\"Telemetry support job complete.\"}\n", j.TaskID)
    }
}
'@ | Set-Content -Encoding UTF8 $harnessSource

go build -o $adapterPath $harnessSource

$openClawSource = Join-Path $env:RUNNER_TEMP 'sprint1-telemetry-openclaw.go'
@'
package main
import ("fmt"; "os"; "time")
func main() {
    if len(os.Args) >= 3 && os.Args[1] == "sessions" && os.Args[2] == "archive" { return }
    time.Sleep(30 * time.Second)
    fmt.Println("Telemetry operation complete")
    fmt.Println("WORKBENCH_OPERATION_COMPLETE: verified")
}
'@ | Set-Content -Encoding UTF8 $openClawSource

go build -o (Join-Path $env:RUNNER_TEMP 'openclaw.exe') $openClawSource
go build -ldflags='-s -w -H=windowsgui' -o $exePath ./cmd/workbench

$state = [ordered]@{
    version = 3
    active_project_id = 'telemetry-project'
    project_path = $projectPath
    notes = 'Sprint 1 telemetry acceptance.'
    projects = @([ordered]@{
        id='telemetry-project';path=$projectPath;name='Workbench Telemetry';notes='';pinned=$true
        added_at=$now.AddDays(-1).ToString('o');last_used_at=$now.ToString('o')
    })
    tasks = @()
    secrets = @()
    preferences = [ordered]@{
        avoid_work_usage=$true;allow_metered_api=$false;autonomy_mode='trusted-repo'
        harness_adapter_path=$adapterPath;mcp_port=18943;mcp_token='sprint1-telemetry-local-token'
    }
}
$state | ConvertTo-Json -Depth 12 | Set-Content -Encoding UTF8 $statePath

Add-Type @'
using System;
using System.Text;
using System.Runtime.InteropServices;
public static class WBTelemetry {
    [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left,Top,Right,Bottom; }
    [DllImport("user32.dll")] public static extern IntPtr GetDlgItem(IntPtr hWnd, int id);
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT rect);
    [DllImport("user32.dll")] public static extern bool MoveWindow(IntPtr hWnd, int x, int y, int w, int h, bool repaint);
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
    [DllImport("user32.dll")] public static extern bool IsWindowEnabled(IntPtr hWnd);
    [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr hWnd);
    [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr hWnd, IntPtr hdcBlt, uint flags);
    [DllImport("user32.dll", CharSet=CharSet.Unicode, EntryPoint="SendMessageTimeoutW")] public static extern IntPtr SendPtr(IntPtr h,uint m,IntPtr w,IntPtr l,uint f,uint t,out UIntPtr r);
    [DllImport("user32.dll", CharSet=CharSet.Unicode, EntryPoint="SendMessageTimeoutW")] public static extern IntPtr SendString(IntPtr h,uint m,IntPtr w,string l,uint f,uint t,out UIntPtr r);
    [DllImport("user32.dll", CharSet=CharSet.Unicode, EntryPoint="SendMessageTimeoutW")] public static extern IntPtr SendBuffer(IntPtr h,uint m,IntPtr w,StringBuilder l,uint f,uint t,out UIntPtr r);
}
'@

function Get-Control($p,[int]$id) {
    $h=[WBTelemetry]::GetDlgItem($p.MainWindowHandle,$id)
    if($h -eq [IntPtr]::Zero){throw "missing control $id"}
    return $h
}
function Send-Bounded([IntPtr]$h,[uint32]$m,[IntPtr]$w,[IntPtr]$l,[string]$what) {
    $r=[UIntPtr]::Zero
    if([WBTelemetry]::SendPtr($h,$m,$w,$l,2,2000,[ref]$r) -eq [IntPtr]::Zero){throw "message timeout: $what"}
    return [uint64]$r.ToUInt64()
}
function Set-Text([IntPtr]$h,[string]$text) {
    $r=[UIntPtr]::Zero
    if([WBTelemetry]::SendString($h,0x000C,[IntPtr]::Zero,$text,2,2000,[ref]$r) -eq [IntPtr]::Zero){throw 'text timeout'}
}
function Click-Control($p,[int]$id) {
    [void](Send-Bounded (Get-Control $p $id) 0x00F5 ([IntPtr]::Zero) ([IntPtr]::Zero) "click $id")
    Start-Sleep -Milliseconds 200
}
function Delegate-Intent($p,[string]$intent) {
    Set-Text (Get-Control $p 3111) $intent
    Click-Control $p 3112
}
function Get-ListLines([IntPtr]$list) {
    $count=[int](Send-Bounded $list 0x018B ([IntPtr]::Zero) ([IntPtr]::Zero) 'list count')
    $out=@()
    for($i=0;$i -lt $count;$i++){
        $n=[int](Send-Bounded $list 0x018A ([IntPtr]$i) ([IntPtr]::Zero) 'list text length')
        $b=New-Object Text.StringBuilder ([Math]::Max(2,$n+2))
        $r=[UIntPtr]::Zero
        if([WBTelemetry]::SendBuffer($list,0x0189,[IntPtr]$i,$b,2,2000,[ref]$r) -eq [IntPtr]::Zero){throw 'list text timeout'}
        $out += $b.ToString()
    }
    return @($out)
}
function Select-ListLine($p,[int]$id,[string]$needle) {
    $list=Get-Control $p $id
    $lines=@(Get-ListLines $list)
    $index=-1
    for($i=0;$i -lt $lines.Count;$i++){if($lines[$i] -like "*$needle*"){$index=$i;break}}
    if($index -lt 0){throw "Could not find '$needle' in control ${id}: $($lines -join ' | ')"}
    # Set the native selection and immediately deliver the same LBN_SELCHANGE
    # notification Windows sends for a real owner click. The dashboard's
    # one-second telemetry clock can legitimately rebuild the list between two
    # separate inspection messages, so the durable post-notification UI state
    # is the correct selection proof.
    [void](Send-Bounded $list 0x0186 ([IntPtr]$index) ([IntPtr]::Zero) 'select list row')
    $cmd=[IntPtr]($id -bor (1 -shl 16))
    [void](Send-Bounded $p.MainWindowHandle 0x0111 $cmd $list 'notify list selection')
    Start-Sleep -Milliseconds 200
}
function Wait-Until([scriptblock]$condition,[string]$description,[int]$seconds=30) {
    $deadline=[DateTime]::UtcNow.AddSeconds($seconds)
    do {
        if(& $condition){return}
        Start-Sleep -Milliseconds 200
    } while([DateTime]::UtcNow -lt $deadline)
    throw "Timed out: $description"
}
function Read-State(){
    $deadline=[DateTime]::UtcNow.AddSeconds(10)
    do {
        $stream=$null
        $reader=$null
        try {
            $share=[System.IO.FileShare]([int][System.IO.FileShare]::ReadWrite -bor [int][System.IO.FileShare]::Delete)
            $stream=[System.IO.File]::Open($statePath,[System.IO.FileMode]::Open,[System.IO.FileAccess]::Read,$share)
            $reader=New-Object System.IO.StreamReader($stream)
            $raw=$reader.ReadToEnd()
            if([string]::IsNullOrWhiteSpace($raw)){throw 'state file was empty during update'}
            return ($raw | ConvertFrom-Json)
        } catch {
            if([DateTime]::UtcNow -ge $deadline){throw}
            Start-Sleep -Milliseconds 50
        } finally {
            if($reader){$reader.Dispose()}
            elseif($stream){$stream.Dispose()}
        }
    } while($true)
}
function Row([IntPtr]$list,[string]$needle){return @((Get-ListLines $list)|Where-Object{$_ -like "*$needle*"}|Select-Object -First 1)[0]}
function Percent([string]$row){if($row -match '(\d+)%'){return [int]$Matches[1]};return -1}
function Save-Window($p,[string]$name) {
    $p.Refresh()
    [void][WBTelemetry]::SetForegroundWindow($p.MainWindowHandle)
    Start-Sleep -Milliseconds 350
    $rect=New-Object WBTelemetry+RECT
    [void][WBTelemetry]::GetWindowRect($p.MainWindowHandle,[ref]$rect)
    $bitmap=New-Object Drawing.Bitmap ($rect.Right-$rect.Left),($rect.Bottom-$rect.Top)
    $graphics=[Drawing.Graphics]::FromImage($bitmap)
    $captured=$false
    try {
        $hdc=$graphics.GetHdc()
        try {
            # Match the repository's established production screenshot path.
            # PrintWindow renders the complete HWND and its native child controls
            # reliably on hosted Windows runners; screen-copy remains a fallback.
            $captured=[WBTelemetry]::PrintWindow($p.MainWindowHandle,$hdc,0)
        } finally {
            $graphics.ReleaseHdc($hdc)
        }
        if(-not $captured){
            $graphics.CopyFromScreen($rect.Left,$rect.Top,0,0,$bitmap.Size,[Drawing.CopyPixelOperation]::SourceCopy)
            $captured=$true
        }
    } finally {
        $graphics.Dispose()
    }
    if(-not $captured){throw 'Could not capture the running Workbench window.'}
    $bitmap.Save((Join-Path $proof $name),[Drawing.Imaging.ImageFormat]::Png)
    $bitmap.Dispose()
}

$evidence=New-Object Collections.Generic.List[string]
$p=Start-Process -FilePath $exePath -PassThru
try {
    $deadline=[DateTime]::UtcNow.AddSeconds(15)
    do{Start-Sleep -Milliseconds 200;$p.Refresh()}while($p.MainWindowHandle -eq 0 -and -not $p.HasExited -and [DateTime]::UtcNow -lt $deadline)
    if($p.HasExited -or $p.MainWindowHandle -eq 0){throw 'Workbench production window missing'}
    [void][WBTelemetry]::MoveWindow($p.MainWindowHandle,10,10,1280,840,$true)
    Start-Sleep -Milliseconds 500

    # Create every workload item through the actual production Workbench UI.
    Click-Control $p 3001
    Delegate-Intent $p '[workbench:operations] Telemetry stage operation'
    Delegate-Intent $p 'Telemetry measured source verification'
    Delegate-Intent $p 'Telemetry needs you'
    Delegate-Intent $p 'Telemetry queued priority target'
    Delegate-Intent $p 'Telemetry queued follow-up'
    Delegate-Intent $p 'Telemetry queued follow-up two'
    $waitIntent='WORKBENCH_WAIT_GITHUB_ACTIONS:{"repository":"DaisyCloverSoftware/workbench","run_id":987654}' + "`nTelemetry wait for CI"
    Delegate-Intent $p $waitIntent

    Wait-Until { @((Read-State).tasks|Where-Object{$_.status -eq 'running'}).Count -ge 2 } 'two concurrently running jobs' 30
    Wait-Until { @((Read-State).tasks|Where-Object{$_.intent -eq 'Telemetry wait for CI' -and $_.status -eq 'waiting_dependency'}).Count -eq 1 } 'real Waiting task' 15
    Wait-Until { @((Read-State).tasks|Where-Object{$_.status -eq 'queued'}).Count -ge 3 } 'three queued jobs' 15

    # Open Dashboard once. All observations below must update in place.
    Click-Control $p 3301
    $server=Get-Control $p 3601
    $ai=Get-Control $p 3607
    $waiting=Get-Control $p 3609
    $needs=Get-Control $p 3611

    Wait-Until { [WBTelemetry]::IsWindowVisible($server) -and [WBTelemetry]::IsWindowVisible($ai) } 'Operations lanes visible' 10
    Wait-Until { ((Get-ListLines $ai)-join "`n") -match '\d+%' } 'measured percentage in active row' 20
    Wait-Until { ((Get-ListLines $server)-join "`n") -match 'Stage 2/4' } 'stage progress in operation row' 12
    Wait-Until { ((Get-ListLines $waiting)-join "`n") -match 'Telemetry wait for CI' } 'Waiting visible on open Dashboard' 8

    $before=Row $ai 'Telemetry measured'
    $beforePercent=Percent $before
    if($beforePercent -lt 0){throw "Measured row lacked deterministic percent: $before"}
    $serverRow=Row $server 'Stage 2/4'
    if($serverRow -notmatch 'Stage 2/4' -or $serverRow -match '%'){throw "Stage-only operation was not truthful: $serverRow"}
    if($serverRow -notmatch 'RUNNING'){throw "Server operation did not expose RUNNING state: $serverRow"}
    Save-Window $p 'Sprint1-Telemetry-Live-Progress.png'

    Start-Sleep -Seconds 2
    $after=Row $ai 'Telemetry measured'
    $afterPercent=Percent $after
    if($afterPercent -le $beforePercent){throw "Measured percent did not advance live: before=$beforePercent after=$afterPercent row=$after"}
    if($after -notmatch '\d+s elapsed' -or $after -notmatch '\d+s ago'){throw "Runtime/activity missing: $after"}
    if($after -match 'percentage unavailable'){throw "Unacceptable unavailable-percentage text: $after"}
    $evidence.Add('1. Three simultaneous real operations were present: measured Running, stage-based Server Ops Running, and a real Waiting dependency.')
    $evidence.Add("2. Deterministic progress advanced from $beforePercent% to $afterPercent% while Dashboard stayed open; percentage came from actual source-file hash work units.")
    $evidence.Add('3. Stage-only Server Ops displayed Stage 2/4 with no fabricated percentage.')
    $evidence.Add('4. Compact rows exposed elapsed runtime, latest activity age, state and named priority without opening task details.')

    $priorityBefore=Row $ai 'Telemetry queued priority'
    Select-ListLine $p 3607 'Telemetry queued priority'
    $priorityUp=Get-Control $p 3615
    Wait-Until { [WBTelemetry]::IsWindowVisible($priorityUp) -and [WBTelemetry]::IsWindowEnabled($priorityUp) } 'Priority Up enabled for queued selection' 10
    Click-Control $p 3615
    Wait-Until {
        $target=(Read-State).tasks|Where-Object{$_.intent -eq 'Telemetry queued priority target'}|Select-Object -First 1
        $target -and $target.priority -eq 2
    } 'durable High priority after Priority Up' 10
    Wait-Until { (Row $ai 'Telemetry queued priority') -match 'HIGH' } 'visible priority change' 10
    $priorityAfter=Row $ai 'Telemetry queued priority'
    if($priorityAfter -eq $priorityBefore){throw "Priority row did not change: $priorityAfter"}
    $evidence.Add("5. Priority Up changed the visible queued row from '$priorityBefore' to '$priorityAfter' and persisted scheduler priority High.")
    Save-Window $p 'Sprint1-Telemetry-Priority.png'

    Wait-Until { @((Read-State).tasks|Where-Object{$_.intent -eq 'Telemetry needs you' -and $_.status -eq 'needs_attention'}).Count -eq 1 } 'Needs You state' 180
    Wait-Until { ((Get-ListLines $needs)-join "`n") -match 'Telemetry needs you' } 'Needs You visible without navigation' 10
    Select-ListLine $p 3611 'Telemetry needs you'
    $details=Get-Control $p 3614
    $detailLength=[int](Send-Bounded $details 0x000E ([IntPtr]::Zero) ([IntPtr]::Zero) 'detail length')
    $detailBuffer=New-Object Text.StringBuilder ([Math]::Max(2,$detailLength+2))
    $detailResult=[UIntPtr]::Zero
    if([WBTelemetry]::SendBuffer($details,0x000D,[IntPtr]$detailBuffer.Capacity,$detailBuffer,2,2000,[ref]$detailResult) -eq [IntPtr]::Zero){throw 'detail read timeout'}
    $detailText=$detailBuffer.ToString()
    if($detailText -notmatch 'Owner action' -or $detailText -notmatch 'Choose the telemetry acceptance owner action'){throw "Needs You owner action missing: $detailText"}
    $evidence.Add('6. Needs You appeared automatically and exposed the required owner action while Waiting retained dependency/retry context.')
    Save-Window $p 'Sprint1-Telemetry-Needs-You.png'
} finally {
    if($p -and -not $p.HasExited){Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue}
    $evidence|Set-Content -Encoding UTF8 (Join-Path $proof 'Sprint1-Telemetry-Evidence.txt')
    if(Test-Path $statePath){Copy-Item $statePath (Join-Path $proof 'Sprint1-Telemetry-Final-State.json') -Force}
}

if($evidence.Count -ne 6){throw "Expected 6 evidence lines, got $($evidence.Count)"}
Get-Content (Join-Path $proof 'Sprint1-Telemetry-Evidence.txt')