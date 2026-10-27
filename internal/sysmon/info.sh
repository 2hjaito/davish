# Prints what the server is, for info.go. Every source is optional.
export LC_ALL=C
echo '@@os'
cat /etc/os-release 2>/dev/null
echo '@@uname'
uname -r
uname -m
echo '@@hostname'
cat /proc/sys/kernel/hostname 2>/dev/null || hostname
echo '@@cpu'
m=$(grep -m 1 '^model name' /proc/cpuinfo 2>/dev/null | cut -d: -f2-)
# ARM kernels often leave the model out of /proc/cpuinfo; lscpu knows it.
[ -z "$m" ] && m=$(lscpu 2>/dev/null | grep -m 1 '^Model name' | cut -d: -f2-)
echo "$m"
echo '@@virt'
systemd-detect-virt 2>/dev/null
echo '@@end'
