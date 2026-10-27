# Prints one process snapshot for procs.go. CPU time comes from
# /proc/<pid>/stat so two snapshots give each process's current load, as
# top shows it; ps only adds the user ID and the full command line.
export LC_ALL=C
echo '@@uptime'
cat /proc/uptime
echo '@@clk'
getconf CLK_TCK 2>/dev/null || echo 100
echo '@@pagesize'
getconf PAGESIZE 2>/dev/null || echo 4096
echo '@@meminfo'
head -n 1 /proc/meminfo
echo '@@passwd'
cut -d: -f1,3 /etc/passwd 2>/dev/null
echo '@@ps'
ps -ww -eo pid=,uid=,args= 2>/dev/null
echo '@@stat'
cat /proc/[0-9]*/stat 2>/dev/null
echo '@@end'
