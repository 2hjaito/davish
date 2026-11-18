---
title: "Monitor"
description: "CPU per core, memory and swap, disks, network per interface, NVIDIA GPUs, processes."
sidebar:
  order: 7
---

CPU per core, memory and swap, disks, network per interface, NVIDIA GPUs, processes. RAM and disk alerts you can tune.

Below the charts, **Overview** describes the machine in four cards: the system (OS, kernel, virtualisation, when it booted, time zone, systemd, who is logged in), the hardware (CPU model, cores and threads, memory, swap), the disks (each drive marked NVMe, SSD, HDD or virtual, with its model, size and what is mounted on it; a mount opens Disk; with Docker, the mount holding Docker's data is tagged and each mount lists the compose projects or containers whose volumes and bind mounts live there) and the network (interfaces sorted LAN, VPN such as Tailscale or WireGuard, then Docker bridges, with their addresses, the default gateway and DNS). Read once with `lsblk`, `ip` and `/proc`; servers without them just show less.

On a Windows server (OpenSSH, Windows PowerShell 5.1, nothing installed) Monitor reads the same figures through PowerShell and CIM: CPU, memory, drives, network, disks and processes. Windows has no load average, I/O wait or steal time, so those are left out. The other apps that need Linux (Services, Docker, Logs, Cron, Nginx…) are not available on Windows servers yet; Launchpad shows them greyed out.
