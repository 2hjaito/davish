# Prints one process snapshot for procs.go, in the shapes procs.sh uses:
# a /proc/<pid>/stat line per process, so two snapshots give each process's
# current load, as top shows it. CPU time comes from the performance counters
# (100 ns units, so clk is 10000000); Win32_Process adds the command line, the
# start time and the memory. Windows PowerShell 5.1 and CIM only.
$ErrorActionPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue'
try { [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false) } catch {}

function N($v) { if ($null -eq $v) { return [int64]0 } else { return [int64]$v } }
function Clean($s) { if ($null -eq $s) { return '' } else { return ($s -replace '[\r\n]+', ' ').Trim() } }

$sys = Get-CimInstance Win32_PerfRawData_PerfOS_System
$freq = [double](N $sys.Frequency_Object)
if ($freq -le 0) { $freq = 1 }
$uptime = ([double](N $sys.Timestamp_Object) - [double](N $sys.SystemUpTime)) / $freq
$os = Get-CimInstance Win32_OperatingSystem
$boot = $os.LastBootUpTime

$cpu = @{}
foreach ($p in @(Get-CimInstance Win32_PerfRawData_PerfProc_Process)) {
	$cpu[[int](N $p.IDProcess)] = N $p.PercentProcessorTime
}
$procs = @(Get-CimInstance Win32_Process)

$sb = New-Object System.Text.StringBuilder
[void]$sb.Append("@@uptime`n$($uptime.ToString('0.00', [Globalization.CultureInfo]::InvariantCulture)) 0.00`n")
[void]$sb.Append("@@clk`n10000000`n")
[void]$sb.Append("@@pagesize`n1`n")
[void]$sb.Append("@@meminfo`nMemTotal: $(N $os.TotalVisibleMemorySize) kB`n")
# Windows has no uid to name table; the owner of each process is not read, as asking costs a call per process.
[void]$sb.Append("@@passwd`n")

[void]$sb.Append("@@ps`n")
foreach ($p in $procs) {
	$id = [int](N $p.ProcessId)
	if ($id -eq 0) { continue } # the idle "process" is not a program
	[void]$sb.Append("$id - $(Clean $p.CommandLine)`n")
}

# Fields 3 to 24 of proc(5): state, ppid, utime (field 14) with stime left at 0,
# threads (20), start in ticks after boot (22), resident size (24, "pages" of 1 byte).
[void]$sb.Append("@@stat`n")
foreach ($p in $procs) {
	$id = [int](N $p.ProcessId)
	if ($id -eq 0) { continue }
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
