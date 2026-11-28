# Prints a metrics snapshot every $every seconds for parse.go, one @@section
# per source, in the same shapes as collect.sh (/proc/stat, /proc/meminfo,
# df -kP…) so the Go side reads both with one parser. Go sets $every and
# $healthEvery above this script. Windows PowerShell 5.1 and CIM only:
# nothing is installed on the server.
$ErrorActionPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue'
try { [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false) } catch {}

# Asking for GPUs and containers only makes sense where their tools exist.
$nvidia = Get-Command nvidia-smi.exe -ErrorAction SilentlyContinue
$docker = Get-Command docker.exe -ErrorAction SilentlyContinue
$sysDrive = $env:SystemDrive
if (-not $sysDrive) { $sysDrive = 'C:' }

# N is a counter as a number; a missing property reads as 0.
function N($v) { if ($null -eq $v) { return [int64]0 } else { return [int64]$v } }

# The first reading of each core's counters, set by the first snapshot.
$script:base = @{}

function L($s) { [void]$script:o.Append("$s`n") }
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

# AMD, Intel and any card without nvidia-smi: Windows counts GPU time and memory
# itself, for every vendor. Looked up once; the load needs the previous reading.
$script:gpus = @(GpuAdapters ([bool]$nvidia))
$script:gpuPrev = @{}
$script:gpuPrevT = [int64]0

function Snapshot($health) {
	$script:o = New-Object System.Text.StringBuilder

	# Uptime first: it is the clock for every rate, so it is read closest to the counters.
	$sys = Get-CimInstance Win32_PerfRawData_PerfOS_System
	$freq = [double](N $sys.Frequency_Object)
	if ($freq -le 0) { $freq = 1 }
	$uptime = ([double](N $sys.Timestamp_Object) - [double](N $sys.SystemUpTime)) / $freq

	# CPU: per core, user / system / idle in 100 ns, summed for the total line.
	# System is everything busy that is not user time (kernel, interrupts, DPCs);
	# the wall clock of the core is Timestamp_Sys100NS, so the three add up to it.
	# That clock counts from the year 1601 and the others from boot, so each
	# counter is shown relative to its first reading: the numbers stay small
	# (a few hundred cores would overflow 64 bits otherwise), and only the
	# differences between snapshots are ever used.
	$cores = @(Get-CimInstance Win32_PerfRawData_PerfOS_Processor | Where-Object { $_.Name -ne '_Total' } |
		Sort-Object { [int]($_.Name -replace '\D', '') })
	$tu = [int64]0; $ts = [int64]0; $ti = [int64]0
	$lines = @()
	$i = 0
	foreach ($c in $cores) {
		if (-not $script:base.ContainsKey($c.Name)) {
			$script:base[$c.Name] = @((N $c.Timestamp_Sys100NS), (N $c.PercentIdleTime), (N $c.PercentUserTime))
		}
		$b = $script:base[$c.Name]
		$wall = [Math]::Max([int64]0, (N $c.Timestamp_Sys100NS) - $b[0])
		$idle = [Math]::Max([int64]0, [Math]::Min((N $c.PercentIdleTime) - $b[1], $wall))
		$user = [Math]::Max([int64]0, [Math]::Min((N $c.PercentUserTime) - $b[2], $wall - $idle))
		$kern = $wall - $idle - $user
		$tu += $user; $ts += $kern; $ti += $idle
		$lines += "cpu$i  $user 0 $kern $idle 0 0 0 0"
		$i++
	}
	L '@@stat'
	L "cpu  $tu 0 $ts $ti 0 0 0 0"
	foreach ($l in $lines) { L $l }

	# Memory, in the units of /proc/meminfo (kB).
	$os = Get-CimInstance Win32_OperatingSystem
	$mem = Get-CimInstance Win32_PerfRawData_PerfOS_Memory
	$totalKB = N $os.TotalVisibleMemorySize
	$availKB = [int64]((N $mem.AvailableBytes) / 1024)
	if ($availKB -le 0) { $availKB = N $os.FreePhysicalMemory }
	$freeKB = [int64]((N $mem.FreeAndZeroPageListBytes) / 1024)
	if ($freeKB -le 0) { $freeKB = N $os.FreePhysicalMemory }
	$cacheKB = [int64](((N $mem.StandbyCacheNormalPriorityBytes) + (N $mem.StandbyCacheReserveBytes) +
		(N $mem.StandbyCacheCoreBytes) + (N $mem.ModifiedPageListBytes)) / 1024)
	$swapTotal = [int64]0; $swapUsed = [int64]0
	foreach ($pf in @(Get-CimInstance Win32_PageFileUsage)) {
		$swapTotal += (N $pf.AllocatedBaseSize) * 1024
		$swapUsed += (N $pf.CurrentUsage) * 1024
	}
	L '@@meminfo'
	L "MemTotal: $totalKB kB"
	L "MemFree: $freeKB kB"
	L "MemAvailable: $availKB kB"
	L 'Buffers: 0 kB'
	L "Cached: $cacheKB kB"
	L "SwapTotal: $swapTotal kB"
	L "SwapFree: $([Math]::Max(0, $swapTotal - $swapUsed)) kB"

	# Windows has no load average; the process count is the one number that carries over.
	L '@@loadavg'
	L "0.00 0.00 0.00 1/$(N $sys.Processes) 0"

	L '@@uptime'
	L ($uptime.ToString('0.00', [Globalization.CultureInfo]::InvariantCulture) + ' 0.00')

	# Network: cumulative bytes per interface, as /proc/net/dev lists them.
	L '@@netdev'
	L 'Inter-|   Receive |  Transmit'
	L ' face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed'
	foreach ($nic in @(Get-CimInstance Win32_PerfRawData_Tcpip_NetworkInterface)) {
		$name = ($nic.Name -replace ':', ' ').Trim()
		if ($name -match '^(isatap|Teredo|6to4|WAN Miniport|Loopback)') { continue }
		L "${name}: $(N $nic.BytesReceivedPersec) 0 0 0 0 0 0 0 $(N $nic.BytesSentPersec) 0 0 0 0 0 0 0"
	}

	# Filesystems as df -kP prints them; the system drive comes first and is the "/".
	L '@@df'
	L 'Filesystem 1024-blocks Used Available Capacity Mounted on'
	$drives = @(Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3' |
		Sort-Object { if ($_.DeviceID -eq $sysDrive) { 0 } else { 1 } }, DeviceID)
	foreach ($d in $drives) {
		$size = N $d.Size
		if ($size -le 0) { continue }
		$free = N $d.FreeSpace
		$used = $size - $free
		$pct = [Math]::Ceiling(100 * $used / $size)
		L "/dev/$($d.DeviceID) $([int64]($size / 1024)) $([int64]($used / 1024)) $([int64]($free / 1024)) $pct% $($d.DeviceID)\"
	}

	# Disk throughput as /proc/diskstats counts it: 512-byte sectors.
	$names = @()
	L '@@diskstats'
	foreach ($pd in @(Get-CimInstance Win32_PerfRawData_PerfDisk_PhysicalDisk | Where-Object { $_.Name -ne '_Total' })) {
		$dn = 'disk' + (($pd.Name -split ' ')[0])
		$names += $dn
		L "0 0 $dn 0 0 $([int64]((N $pd.DiskReadBytesPersec) / 512)) 0 0 0 $([int64]((N $pd.DiskWriteBytesPersec) / 512))"
	}
	L '@@block'
	foreach ($dn in $names) { L $dn }

	L '@@gpu'
	if ($nvidia) {
		foreach ($g in @(& nvidia-smi.exe --query-gpu=index,name,utilization.gpu,memory.used,memory.total,temperature.gpu,power.draw --format=csv,noheader,nounits 2>$null)) { L $g }
	}

	# One line per card, as collect.sh prints them: slot|vendor|driver|busy %|vram used|vram total|temp|power|name
	L '@@gpudrm'
	if ($script:gpus.Count -gt 0) {
		$cur = @{}
		$t = [int64]0
		foreach ($e in @(Get-CimInstance Win32_PerfRawData_GPUPerformanceCounters_GPUEngine)) {
			if ($e.Name -match '(luid_0x[0-9A-Fa-f]+_0x[0-9A-Fa-f]+)_phys_\d+_eng_(\d+)_engtype_(\w+)') {
				$k = "$($e.Name -replace '^pid_\d+_', '' -replace '_eng_.*$', '')/$($Matches[2])/$($Matches[3])"
				$cur[$k] = [int64]($cur[$k]) + [int64](N $e.UtilizationPercentage)
				$t = [Math]::Max($t, (N $e.Timestamp_Sys100NS))
			}
		}
		$adapterMem = @(Get-CimInstance Win32_PerfRawData_GPUPerformanceCounters_GPUAdapterMemory)
		foreach ($a in $script:gpus) {
			# The busiest engine of the card, as Task Manager's GPU column shows it.
			$busy = ''
			if ($script:gpuPrevT -gt 0 -and $t -gt $script:gpuPrevT) {
				$best = [double]0
				foreach ($k in $cur.Keys) {
					$mine = ($a.Luid -eq '*') -or ($a.Luid -and $k.StartsWith($a.Luid))
					if ($mine -and $script:gpuPrev.ContainsKey($k)) {
						$d = [double]($cur[$k] - $script:gpuPrev[$k]) / [double]($t - $script:gpuPrevT) * 100
						if ($d -gt $best) { $best = $d }
					}
				}
				$busy = [string][Math]::Round([Math]::Min($best, 100), 1, 'AwayFromZero')
			}
			$used = ''
			foreach ($m in $adapterMem) {
				if ((AdapterFor @($a) $m.Name)) { $used = [string](N $m.DedicatedUsage); break }
			}
			$total = ''
			if ($a.Total -gt 0) { $total = [string]$a.Total }
			L "$($a.Slot)|$($a.Vendor)|wddm|$($busy.Replace(',', '.'))|$used|$total|||$($a.Name)"
		}
		$script:gpuPrev = $cur
		$script:gpuPrevT = $t
	}
	L '@@gpuname'

	if ($health) {
		# Services set to start by themselves that stopped with an error. 1077
		# means "never started", which trigger-started services report all the time.
		L '@@failed'
		foreach ($s in @(Get-CimInstance Win32_Service -Filter "StartMode='Auto' AND State<>'Running'")) {
			$code = N $s.ExitCode
			if ($code -ne 0 -and $code -ne 1077) { L "$($s.Name).service" }
		}
		L '@@containers'
		if ($docker) {
			$ids = @(& docker.exe ps -aq 2>$null)
			if ($ids.Count -gt 0) {
				foreach ($c in @(& docker.exe inspect --format '{{.Name}} {{.RestartCount}} {{.State.Status}}' @ids 2>$null)) { L $c }
			}
		}
	}
	L '@@end'
	return $script:o.ToString()
}

# Nothing in here ends the loop when Davish goes away. Tried on Windows 11 with
# OpenSSH, none of the signals a script could wait for arrives: stdin never
# ends and nothing written to it after the script is delivered (so no heartbeat
# can work), and a write to a closed stdout may not throw, as .NET's console
# streams ignore a broken pipe. The catch below is kept for the servers where
# it does. Whether sshd itself ends the process when the channel closes is what
# keeps this from leaving one behind.
$n = 0
while ($true) {
	$text = Snapshot (($n % $healthEvery) -eq 0)
	try {
		[Console]::Out.Write($text)
		[Console]::Out.Flush()
	} catch {
		[Environment]::Exit(0)
	}
	$n++
	Start-Sleep -Seconds $every
}
