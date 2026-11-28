---
title: "Monitor"
description: "CPU per core, memory and swap, disks, network per interface, GPUs (NVIDIA, AMD, Intel), processes."
sidebar:
  order: 7
---

CPU per core, memory and swap, disks, network per interface, GPUs (NVIDIA, AMD, Intel), processes. RAM and disk alerts you can tune.

Below the charts, **Overview** describes the machine in four cards: the system (OS, kernel, virtualisation, when it booted, time zone, systemd, who is logged in), the hardware (CPU model, cores and threads, memory, swap), the disks (each drive marked NVMe, SSD, HDD or virtual, with its model, size and what is mounted on it; a mount opens Disk; with Docker, the mount holding Docker's data is tagged and each mount lists the compose projects or containers whose volumes and bind mounts live there) and the network (interfaces sorted LAN, VPN such as Tailscale or WireGuard, then Docker bridges, with their addresses, the default gateway and DNS). Read once with `lsblk`, `ip` and `/proc`; servers without them just show less.

Below the GPU cards, a **Processes** table lists the processes on the GPUs, like nvtop: PID, user, GPU, type (compute or graphics), utilization, GPU memory and its share of the card, CPU, host memory and the command line, sorted by any column. On NVIDIA it reads `nvidia-smi` (`--query-compute-apps` and a one-second `pmon` sample) and `/proc` for just those processes, every 3 seconds and only while the GPU tab is open. Where the driver has no `pmon` (MIG mode, some older drivers) only compute processes show, without utilization; Windows in WDDM mode does not report GPU memory per process; a process `nvidia-smi` sees but the login does not (`nvidia-smi` in a container) keeps its name, without user, CPU or host memory. AMD and Intel GPUs have no `nvidia-smi`: on Linux Davish reads the cards from `/sys/class/drm` (AMD: load, GPU memory, temperature, power; Intel: the name only, as its driver gives no load there) and each process's GPU time and memory from the DRM descriptors in `/proc/PID/fdinfo`, the same source nvtop uses, so the utilization column fills in from the second reading and a login that is not root sees only its own processes. On Windows every vendor comes from the GPU performance counters (engine load, memory per card and per process), without temperature or power.

On a Windows server (OpenSSH, Windows PowerShell 5.1, nothing installed) Monitor reads the same figures through PowerShell and CIM: CPU, memory, drives, network, disks and processes. Windows has no load average, I/O wait or steal time, so those are left out. The other apps that need Linux (Services, Docker, Logs, Cron, Nginx…) are not available on Windows servers yet; Launchpad shows them greyed out.
