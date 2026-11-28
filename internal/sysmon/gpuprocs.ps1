# Prints the processes on GPUs for gpuprocs.go, in the shapes gpuprocs.sh
# uses: nvidia-smi's two views of the NVIDIA ones, the other cards, and what
# Windows counts per process on them (written like the DRM descriptors of
# Linux, so one parser reads both), then the sections of procs.ps1 for just
# those PIDs. Windows PowerShell 5.1 and CIM only.
$ErrorActionPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue'
try { [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false) } catch {}

function N($v) { if ($null -eq $v) { return [int64]0 } else { return [int64]$v } }
function Clean($s) { if ($null -eq $s) { return '' } else { return ($s -replace '[\r\n|]+', ' ').Trim() } }

# The video adapters nvidia-smi does not cover (all of them when it is missing),
# with the LUID the GPU performance counters name them by. Real adapters only:
# a remote desktop or the basic display driver has no PCI ID.
function GpuAdapters($skipNvidia) {
	$out = @()
	$pci = @(Get-CimInstance Win32_VideoController | Where-Object { $_.PNPDeviceID -like 'PCI*' })
	$classKey = 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}'
	$mask = [int64]4294967295
	$i = 0
	foreach ($vc in $pci) {
		$i++
		$ven = ''
		if ($vc.PNPDeviceID -match 'VEN_([0-9A-Fa-f]{4})') { $ven = '0x' + $Matches[1].ToLower() }
		if ($ven -eq '0x10de' -and $skipNvidia) { continue }
		# The LUID is a device property (Windows 10 1709 and later). With a single adapter
		# the counters can only be its own, so "*" stands for any LUID.
		$luid = ''
		$v = (Get-PnpDeviceProperty -InstanceId $vc.PNPDeviceID -KeyName 'DEVPKEY_Gpu_Luid' -ErrorAction SilentlyContinue).Data
		if ($null -ne $v) {
			$u = [int64]$v
			$luid = 'luid_0x{0:X8}_0x{1:X8}' -f (($u -shr 32) -band $mask), ($u -band $mask)
		} elseif ($pci.Count -eq 1) {
			$luid = '*'
		}
		# AdapterRAM is 32 bits and shows 4 GB for a bigger card; the driver's own value does not.
		$total = [int64](N $vc.AdapterRAM)
		foreach ($k in @(Get-ChildItem $classKey -ErrorAction SilentlyContinue)) {
			$p = Get-ItemProperty $k.PSPath -ErrorAction SilentlyContinue
			if ($p.DriverDesc -eq $vc.Name -and $p.'HardwareInformation.qwMemorySize') {
				$total = [int64]$p.'HardwareInformation.qwMemorySize'
				break
			}
		}
		$slot = $luid
		if (-not $slot) { $slot = "vc$i" }
		$out += [pscustomobject]@{ Slot = $slot; Luid = $luid; Vendor = $ven; Name = (Clean $vc.Name); Total = $total }
	}
	return $out
}

# The adapter an instance name such as "pid_4_luid_0x00000000_0x0000D3F2_phys_0..." belongs to.
function AdapterFor($adapters, $inst) {
	foreach ($a in $adapters) {
		if ($a.Luid -eq '*' -or ($a.Luid -and $inst -like "*$($a.Luid)*")) { return $a }
	}
	return $null
}

$gpus = @()
$apps = @()
$pmon = @()
$nvidia = [bool](Get-Command nvidia-smi.exe -ErrorAction SilentlyContinue)
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

# Windows counts what each process used of each card: engine running time
# (cumulative, in 100 ns) and dedicated memory. Only the cards above have them read.
$cards = @(GpuAdapters $nvidia)
$drmfd = @()
if ($cards.Count -gt 0) {
	$use = @{}
	foreach ($e in @(Get-CimInstance Win32_PerfRawData_GPUPerformanceCounters_GPUEngine)) {
		if ($e.Name -match '^pid_(\d+)_.*_engtype_(\w+)$') {
			$a = AdapterFor $cards $e.Name
			$ns = [int64](N $e.UtilizationPercentage) * 100
			if ($a -and $ns -gt 0) {
				$k = "$($Matches[1])|$($a.Slot)"
				if (-not $use.ContainsKey($k)) { $use[$k] = @{ eng = @{}; mem = [int64]-1 } }
				$type = ($Matches[2].ToLower() -replace '[^a-z0-9_-]', '')
				$use[$k].eng[$type] = [int64]($use[$k].eng[$type]) + $ns
			}
		}
	}
	foreach ($m in @(Get-CimInstance Win32_PerfRawData_GPUPerformanceCounters_GPUProcessMemory)) {
		if ($m.Name -match '^pid_(\d+)_') {
			$a = AdapterFor $cards $m.Name
			$bytes = [int64](N $m.DedicatedUsage)
			if ($a -and $bytes -gt 0) {
				$k = "$($Matches[1])|$($a.Slot)"
				if (-not $use.ContainsKey($k)) { $use[$k] = @{ eng = @{}; mem = [int64]-1 } }
				$use[$k].mem = [Math]::Max([int64]0, $use[$k].mem) + $bytes
			}
		}
	}
	$fdn = 0
	foreach ($k in $use.Keys) {
		$pidSlot = $k -split '\|', 2
		$pids[[int]$pidSlot[0]] = $true
		$fdn++
		$base = "/proc/$($pidSlot[0])/fdinfo/${fdn}:"
		$drmfd += "${base}drm-driver:`twddm"
		$drmfd += "${base}drm-pdev:`t$($pidSlot[1])"
		foreach ($type in $use[$k].eng.Keys) { $drmfd += "${base}drm-engine-${type}:`t$($use[$k].eng[$type]) ns" }
		if ($use[$k].mem -ge 0) { $drmfd += "${base}drm-memory-vram:`t$($use[$k].mem) B" }
	}
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
[void]$sb.Append("@@drmcards`n")
foreach ($c in $cards) { [void]$sb.Append("$($c.Slot)|$($c.Vendor)|$(if ($c.Total -gt 0) { $c.Total })`n") }
[void]$sb.Append("@@drmfd`n")
foreach ($l in $drmfd) { [void]$sb.Append("$l`n") }
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
