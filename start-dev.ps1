$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$backendDir = Join-Path $repoRoot 'backend'
$frontendDir = Join-Path $repoRoot 'frontend'
$logDir = Join-Path $repoRoot '.dev-logs'
$pidFile = Join-Path $logDir 'launcher.pid'
$frontendPort = 4173
$backendPort = 8080

if (-not (Test-Path $logDir)) {
  New-Item -ItemType Directory -Path $logDir | Out-Null
}

Add-Type @"
using System;
using System.Runtime.InteropServices;

public static class JobObjectNative
{
    [StructLayout(LayoutKind.Sequential)]
    public struct JOBOBJECT_BASIC_LIMIT_INFORMATION
    {
        public long PerProcessUserTimeLimit;
        public long PerJobUserTimeLimit;
        public uint LimitFlags;
        public UIntPtr MinimumWorkingSetSize;
        public UIntPtr MaximumWorkingSetSize;
        public uint ActiveProcessLimit;
        public UIntPtr Affinity;
        public uint PriorityClass;
        public uint SchedulingClass;
    }

    [StructLayout(LayoutKind.Sequential)]
    public struct IO_COUNTERS
    {
        public ulong ReadOperationCount;
        public ulong WriteOperationCount;
        public ulong OtherOperationCount;
        public ulong ReadTransferCount;
        public ulong WriteTransferCount;
        public ulong OtherTransferCount;
    }

    [StructLayout(LayoutKind.Sequential)]
    public struct JOBOBJECT_EXTENDED_LIMIT_INFORMATION
    {
        public JOBOBJECT_BASIC_LIMIT_INFORMATION BasicLimitInformation;
        public IO_COUNTERS IoInfo;
        public UIntPtr ProcessMemoryLimit;
        public UIntPtr JobMemoryLimit;
        public UIntPtr PeakProcessMemoryUsed;
        public UIntPtr PeakJobMemoryUsed;
    }

    [DllImport("kernel32.dll", CharSet = CharSet.Unicode)]
    public static extern IntPtr CreateJobObject(IntPtr lpJobAttributes, string lpName);

    [DllImport("kernel32.dll", SetLastError = true)]
    [return: MarshalAs(UnmanagedType.Bool)]
    public static extern bool SetInformationJobObject(
        IntPtr hJob,
        int JobObjectInfoClass,
        IntPtr lpJobObjectInfo,
        uint cbJobObjectInfoLength);

    [DllImport("kernel32.dll", SetLastError = true)]
    [return: MarshalAs(UnmanagedType.Bool)]
    public static extern bool AssignProcessToJobObject(IntPtr hJob, IntPtr hProcess);
}
"@

function New-KillOnCloseJob {
  $job = [JobObjectNative]::CreateJobObject([IntPtr]::Zero, "WhaleVaultDev-$PID")
  if ($job -eq [IntPtr]::Zero) {
    throw "Unable to create Windows Job Object."
  }

  $info = New-Object JobObjectNative+JOBOBJECT_EXTENDED_LIMIT_INFORMATION
  $info.BasicLimitInformation.LimitFlags = 0x2000

  $length = [System.Runtime.InteropServices.Marshal]::SizeOf($info)
  $pointer = [System.Runtime.InteropServices.Marshal]::AllocHGlobal($length)

  try {
    [System.Runtime.InteropServices.Marshal]::StructureToPtr($info, $pointer, $false)
    $ok = [JobObjectNative]::SetInformationJobObject($job, 9, $pointer, [uint32]$length)
    if (-not $ok) {
      throw "Unable to configure Windows Job Object."
    }
  }
  finally {
    [System.Runtime.InteropServices.Marshal]::FreeHGlobal($pointer)
  }

  return $job
}

function Test-PortFree {
  param([int]$Port)

  $listening = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  if ($listening) {
    $owners = ($listening | Select-Object -ExpandProperty OwningProcess -Unique) -join ', '
    throw "Port $Port is already in use by process id(s): $owners"
  }
}

function Start-ManagedProcess {
  param(
    [Parameter(Mandatory = $true)][IntPtr]$JobHandle,
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][string]$WorkingDirectory,
    [Parameter(Mandatory = $true)][string]$Command,
    [Parameter(Mandatory = $true)][string]$StdOutPath,
    [Parameter(Mandatory = $true)][string]$StdErrPath
  )

  if (Test-Path $StdOutPath) { Remove-Item $StdOutPath -Force }
  if (Test-Path $StdErrPath) { Remove-Item $StdErrPath -Force }

  $process = Start-Process `
    -FilePath 'powershell.exe' `
    -WorkingDirectory $WorkingDirectory `
    -ArgumentList @('-NoProfile', '-Command', $Command) `
    -RedirectStandardOutput $StdOutPath `
    -RedirectStandardError $StdErrPath `
    -WindowStyle Hidden `
    -PassThru

  if (-not [JobObjectNative]::AssignProcessToJobObject($JobHandle, $process.Handle)) {
    try { Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue } catch {}
    throw "Unable to attach $Name process to the Windows Job Object."
  }

  return $process
}

function Start-Watchdog {
  param(
    [Parameter(Mandatory = $true)][int]$LauncherPid,
    [Parameter(Mandatory = $true)][int[]]$ChildPids,
    [Parameter(Mandatory = $true)][string]$PidFilePath
  )

  $childPidList = ($ChildPids | ForEach-Object { $_.ToString() }) -join ','
  $command = @"
`$launcherPid = $LauncherPid
`$childPids = @($childPidList)
Write-Output "watchdog-start launcher=`$launcherPid children=`$(`$childPids -join ',')"
while (Get-Process -Id `$launcherPid -ErrorAction SilentlyContinue) {
  Start-Sleep -Seconds 1
}
Write-Output "watchdog-detected-launcher-exit"
foreach (`$childPid in `$childPids) {
  try {
    Write-Output "watchdog-killing `$childPid"
    Start-Process -FilePath 'taskkill.exe' -ArgumentList '/PID', `$childPid, '/T', '/F' -WindowStyle Hidden -Wait | Out-Null
  }
  catch {}
}
if (Test-Path '$PidFilePath') {
  Remove-Item '$PidFilePath' -Force -ErrorAction SilentlyContinue
}
Write-Output "watchdog-finished"
"@

  return Start-Process `
    -FilePath 'powershell.exe' `
    -ArgumentList @('-NoProfile', '-WindowStyle', 'Hidden', '-Command', $command) `
    -RedirectStandardOutput (Join-Path $logDir 'watchdog.out.log') `
    -RedirectStandardError (Join-Path $logDir 'watchdog.err.log') `
    -WindowStyle Hidden `
    -PassThru
}

function Wait-ForHttp {
  param(
    [Parameter(Mandatory = $true)][string]$Url,
    [Parameter(Mandatory = $true)][string]$Name,
    [int]$TimeoutSeconds = 60
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  do {
    try {
      $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 3
      if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 500) {
        return
      }
    }
    catch {
      Start-Sleep -Milliseconds 800
    }
  } while ((Get-Date) -lt $deadline)

  throw "$Name did not become healthy in time. Check logs under $logDir"
}

$jobHandle = $null
$frontendProcess = $null
$backendProcess = $null
$watchdogProcess = $null

try {
  Set-Content -Path $pidFile -Value $PID

  Test-PortFree -Port $frontendPort
  Test-PortFree -Port $backendPort

  $jobHandle = New-KillOnCloseJob

  $backendProcess = Start-ManagedProcess `
    -JobHandle $jobHandle `
    -Name 'backend' `
    -WorkingDirectory $backendDir `
    -Command 'go run .' `
    -StdOutPath (Join-Path $logDir 'backend.out.log') `
    -StdErrPath (Join-Path $logDir 'backend.err.log')

  $frontendProcess = Start-ManagedProcess `
    -JobHandle $jobHandle `
    -Name 'frontend' `
    -WorkingDirectory $frontendDir `
    -Command 'npm run dev -- --host 0.0.0.0 --port 4173' `
    -StdOutPath (Join-Path $logDir 'frontend.out.log') `
    -StdErrPath (Join-Path $logDir 'frontend.err.log')

  $watchdogProcess = Start-Watchdog `
    -LauncherPid $PID `
    -ChildPids @($frontendProcess.Id, $backendProcess.Id) `
    -PidFilePath $pidFile

  Wait-ForHttp -Url "http://127.0.0.1:$backendPort/healthz" -Name 'Backend'
  Wait-ForHttp -Url "http://127.0.0.1:$frontendPort" -Name 'Frontend'

  Write-Host ''
  Write-Host 'Whale Vault dev environment is ready.' -ForegroundColor Green
  Write-Host "Frontend: http://localhost:$frontendPort"
  Write-Host "Backend : http://127.0.0.1:$backendPort/healthz"
  Write-Host "Logs    : $logDir"
  Write-Host ''
  Write-Host 'Closing this script window will automatically stop both frontend and backend.' -ForegroundColor Yellow
  Write-Host 'Press Enter to stop them manually.' -ForegroundColor Yellow

  [void](Read-Host)
}
finally {
  if (Test-Path $pidFile) {
    Remove-Item $pidFile -Force -ErrorAction SilentlyContinue
  }

  foreach ($process in @($frontendProcess, $backendProcess, $watchdogProcess)) {
    if ($null -ne $process) {
      try {
        if (-not $process.HasExited) {
          Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
      }
      catch {}
    }
  }
}
