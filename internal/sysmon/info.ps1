# Prints what the server is, for info.go, in the shapes info.sh uses.
# Every source is optional. Windows PowerShell 5.1 and CIM only.
$ErrorActionPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue'
try { [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false) } catch {}

$os = Get-CimInstance Win32_OperatingSystem
$cs = Get-CimInstance Win32_ComputerSystem
$cpu = @(Get-CimInstance Win32_Processor)[0]

$caption = ("$($os.Caption)" -replace '^Microsoft\s+', '').Trim()
if (-not $caption) { $caption = 'Windows' }
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
	'AMD64' { 'x86_64' }
	'ARM64' { 'aarch64' }
	'x86' { 'i686' }
	default { "$env:PROCESSOR_ARCHITECTURE" }
}

# Which hypervisor, as systemd-detect-virt would name it; "none" on bare metal.
$maker = "$($cs.Manufacturer)"
$model = "$($cs.Model)"
$virt = 'none'
if ($maker -match 'Microsoft' -and $model -match 'Virtual') { $virt = 'microsoft' }
elseif ($maker -match 'VMware' -or $model -match 'VMware') { $virt = 'vmware' }
elseif ($maker -match 'QEMU' -or $model -match 'KVM|QEMU') { $virt = 'kvm' }
elseif ($model -match 'VirtualBox') { $virt = 'oracle' }
elseif ($maker -match 'Xen' -or $model -match 'HVM domU') { $virt = 'xen' }
elseif ($maker -match 'Google') { $virt = 'google' }
elseif ($maker -match 'Amazon' -or $model -match 'EC2') { $virt = 'amazon' }

$sb = New-Object System.Text.StringBuilder
[void]$sb.Append("@@os`nPRETTY_NAME=`"$caption`"`n")
[void]$sb.Append("@@uname`n$($os.Version)`n$arch`n")
[void]$sb.Append("@@hostname`n$env:COMPUTERNAME`n")
[void]$sb.Append("@@cpu`n$(("$($cpu.Name)").Trim())`n")
[void]$sb.Append("@@virt`n$virt`n")
[void]$sb.Append("@@end`n")
[Console]::Out.Write($sb.ToString())
