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

# The loop ends with the session: when stdout is closed the write throws, and
# a server never gets a collector left running behind a closed Davish.
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
