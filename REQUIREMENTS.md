# CESO / ESXi Commander Requirements (Ubuntu-only)

## Goal
A small, auditable service that lets Claude Code manage a standalone VMware ESXi host over SSH to **create, clone, back up, and restore Ubuntu LTS virtual machines**.

## Primary Use Cases
1. **Create VM from a golden Ubuntu template**
   - Provide static IPv4/IPv6, hostname, and SSH keys via **cloud-init guestinfo**.

2. **Full clone of an existing VM**
   - Cold (preferred) or snapshot-based hot clone.
   - Re-IP and rename using cloud-init injection.

3. **Reconfigure an existing VM**
   - CPUs, RAM, disks, NICs, firmware (BIOS/UEFI), Secure Boot.
   - Attach/detach ISOs, adjust NIC adapter, change portgroups.

4. **Back up a VM**
   - Cold backup or snapshot-based hot backup.
   - Compression (zstd), optional encryption (age/PGP).
   - Targets: datastore, NFS/SFTP, S3 (via bridge host).
   - Retention policies (keep_last, keep_daily, keep_weekly, keep_monthly).

5. **Restore a VM**
   - As new or in place.
   - Ensure unique UUID/MAC and re-IP.

6. **Audit and dry-runs**
   - Show exact ESXi commands before running.
   - Immutable, structured logs with secret redaction.

## Scope
- Single ESXi 7.x/8.x host (no vCenter).
- Golden images: **Ubuntu LTS with cloud-init + open-vm-tools preinstalled**.
- Interfaces: CLI (primary), optional local HTTP/JSON API (off by default).
- Out of scope: Windows/Cloudbase-Init, vCenter-only features, GUI.

## Success Metrics
- Create VM: ≤ 90 s (excluding OS boot).
- Clone 80 GB thin VM: ≤ 5 min on SSD datastore.
- Backup error rate < 1% over 30 days; restore success ≥ 99%.
- Zero manual steps in ESXi UI.

## Security
- IP allowlists.
- Short-lived SSH certs for operators.
- Append-only logs, optionally shipped off-box.
- Secrets stored in OS keyring or encrypted files; never logged.

## Rollout Plan
- **Milestone 1 (MVP):** Discovery, Create (Ubuntu), Cold Clone, Cold Backup/Restore, CLI, Audit Logs.
- **Milestone 2:** Hot backup, reconfigure matrix (CPU/RAM/disk/NIC), Secure Boot toggles, IPAM pools, DNS plugin.
- **Milestone 3:** S3/NFS/SFTP targets, local HTTP API, advanced extraConfig.
- **Milestone 4:** PCI passthrough and additional advanced options.

## Validation & Testing
- Unit: command builders, config validators.
- Integration: VM boots with correct hostname/IP; clone uniqueness; backup/restore boots.
- Chaos: datastore full, snapshot failures, network blips.

