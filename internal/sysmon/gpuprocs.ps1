# Prints the processes on NVIDIA GPUs for gpuprocs.go, in the shapes
# gpuprocs.sh uses: nvidia-smi's two views of them, then the sections of
# procs.ps1 for just their PIDs. Windows PowerShell 5.1 and CIM only.
$ErrorActionPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue'
try { [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false) } catch {}

function N($v) { if ($null -eq $v) { return [int64]0 } else { return [int64]$v } }
function Clean($s) { if ($null -eq $s) { return '' } else { return ($s -replace '[\r\n]+', ' ').Trim() } }

$gpus = @()
$apps = @()
$pmon = @()
if (Get-Command nvidia-smi.exe -ErrorAction SilentlyContinue) {
	$gpus = @(& nvidia-smi.exe --query-gpu=index,uuid,memory.total --format=csv,noheader,nounits 2>$null)
	$apps = @(& nvidia-smi.exe --query-compute-apps=gpu_uuid,pid,used_gpu_memory,process_name --format=csv,noheader,nounits 2>$null)
	# pmon samples for about a second, where the driver has it.
	$pmon = @(& nvidia-smi.exe pmon -c 1 -s um 2>$null)
}

# Every PID either view names; an idle GPU's pmon row has "-".
$pids = @{}
foreach ($a in $apps) {
	$f = "$a" -split ','
	$n = 0
	if ($f.Count -ge 2 -and [int]::TryParse($f[1].Trim(), [ref]$n)) { $pids[$n] = $true }
}
foreach ($l in $pmon) {
	if ("$l".TrimStart().StartsWith('#')) { continue }
	$f = -split "$l"
	$n = 0
	if ($f.Count -ge 2 -and [int]::TryParse($f[1], [ref]$n)) { $pids[$n] = $true }
}

$sys = Get-CimInstance Win32_PerfRawData_PerfOS_System
$freq = [double](N $sys.Frequency_Object)
if ($freq -le 0) { $freq = 1 }
$uptime = ([double](N $sys.Timestamp_Object) - [double](N $sys.SystemUpTime)) / $freq
$os = Get-CimInstance Win32_OperatingSystem
$boot = $os.LastBootUpTime

$procs = @()
$cpu = @{}
if ($pids.Count -gt 0) {
	$procs = @(Get-CimInstance Win32_Process -Filter ((@($pids.Keys) | ForEach-Object { "ProcessId=$_" }) -join ' OR '))
	foreach ($p in @(Get-CimInstance Win32_PerfRawData_PerfProc_Process -Filter ((@($pids.Keys) | ForEach-Object { "IDProcess=$_" }) -join ' OR '))) {
		$cpu[[int](N $p.IDProcess)] = N $p.PercentProcessorTime
	}
}

$sb = New-Object System.Text.StringBuilder
[void]$sb.Append("@@gpus`n")
foreach ($l in $gpus) { [void]$sb.Append("$l`n") }
[void]$sb.Append("@@apps`n")
foreach ($l in $apps) { [void]$sb.Append("$l`n") }
[void]$sb.Append("@@pmon`n")
foreach ($l in $pmon) { [void]$sb.Append("$l`n") }
[void]$sb.Append("@@uptime`n$($uptime.ToString('0.00', [Globalization.CultureInfo]::InvariantCulture)) 0.00`n")
[void]$sb.Append("@@clk`n10000000`n")
[void]$sb.Append("@@pagesize`n1`n")
[void]$sb.Append("@@meminfo`nMemTotal: $(N $os.TotalVisibleMemorySize) kB`n")
# Windows has no uid to name table; the owner of each process is not read, as asking costs a call per process.
[void]$sb.Append("@@passwd`n")

[void]$sb.Append("@@ps`n")
foreach ($p in $procs) {
	[void]$sb.Append("$([int](N $p.ProcessId)) - $(Clean $p.CommandLine)`n")
}

# The stat line of procs.ps1: fields 3 to 24 of proc(5), CPU time in 100 ns,
# start in 100 ns after boot, resident size in "pages" of 1 byte.
[void]$sb.Append("@@stat`n")
foreach ($p in $procs) {
	$id = [int](N $p.ProcessId)
	$start = [int64]0
	if ($p.CreationDate -and $boot) { $start = [Math]::Max([int64]0, [int64](($p.CreationDate - $boot).TotalSeconds * 10000000)) }
	$ticks = [int64]0
	if ($cpu.ContainsKey($id)) { $ticks = $cpu[$id] }
	$name = (Clean $p.Name) -replace '[\(\)]', '_'
	if (-not $name) { $name = 'unknown' }
	[void]$sb.Append("$id ($name) S $(N $p.ParentProcessId) 0 0 0 0 0 0 0 0 0 $ticks 0 0 0 0 0 $(N $p.ThreadCount) 0 $start 0 $(N $p.WorkingSetSize)`n")
}
[void]$sb.Append("@@end`n")
[Console]::Out.Write($sb.ToString())
