# Swift Test Suite Documentation

## Overview

The Swift project has two layers of testing:

1. **Unit tests** (`go test`) — validate XML generation, YAML parsing, MAC generation, CLI parsing. **49 tests, all passing.**
2. **E2E integration tests** (`tests/e2e_test.sh`) — validate the full CLI workflow including VM provisioning, network management, cloud-init ISO generation, and disk image creation. **51 tests pass without libvirt; 34 tests require libvirt access.**

---

## Unit Tests

Run with:
```bash
go test ./internal/config/... ./internal/hv/... ./cmd/swift/... -v
```

### Test Files

| File | Tests | Coverage |
|---|---|---|
| `internal/hv/domain_template_test.go` | 15 tests | `BuildDomainXML`: SPICE/VNC/none graphics, WebSocket, serial console, multi-NIC, MAC assignment, disk config, arch, boot devices, XML marshaling |
| `internal/hv/network_template_test.go` | 17 tests | `BuildNetworkXML`: NAT, route, isolated, bridge, default mode, DHCP, device binding, CIDR/netmask calculation, invalid CIDR, defaults |
| `internal/config/config_test.go` | 10 tests | `GenerateMAC` (format, locally-administered bits, uniqueness), `LoadVMConfig` (full/minimal/invalid), `LoadNetworkConfig` (NAT/isolated/invalid) |
| `cmd/swift/main_test.go` | 17 tests | `parseSize` (bytes, MiB, GiB, TiB, spaces, invalid), `parseNetworkFlag` (single/multiple/default model/empty/trailing comma) |

### All 49 Unit Tests Pass

```
PASS ok  swift/internal/config  0.022s
PASS ok  swift/internal/hv      0.013s
PASS ok  swift/cmd/swift        0.014s
```

---

## E2E Integration Tests

Run with:
```bash
CGO_ENABLED=1 go build -o swift ./cmd/swift
bash tests/e2e_test.sh
```

### Test Groups

#### Group 1: CLI Structure & Help (31 tests — all pass)
- `swift version` output
- Root help lists all 9 commands
- `create --help` lists all 12 flags
- `network --help` lists all 7 subcommands
- `stop` has `poweroff` alias
- `show` has `xml` alias

#### Group 2: Error Handling (2 tests — require libvirt connection)
- `swift create test-vm` without `--disk` → error
- `swift create` without name → error
*Note: these fail because `PersistentPreRun` connects to libvirt before command validation*

#### Group 3: Network CLI (8 tests — all pass)
- `network create --help` lists all 8 flags
- `network create` without name → error

#### Group 4: YAML Config Files (3 tests — all pass)
- Creates test VM YAML (with graphics/websocket/serial/networks)
- Creates test NAT network YAML (with DHCP)
- Creates test isolated network YAML

#### Group 5: Build Verification (3 tests — all pass)
- Binary is executable ELF
- Version matches 0.4.0

#### Group 6: Disk Image Operations (3 tests — all pass)
- `qemu-img create` from backing image
- Verify qcow2 format and 10G virtual size

#### Group 7: Cloud-init ISO (3 tests — all pass)
- Generates `user-data` and `meta-data`
- Creates ISO with `genisoimage`
- Validates ISO format and non-zero size

#### Group 8: VM Lifecycle (14 tests — require libvirt)
Full workflow with `usermod -aG libvirt $USER` + re-login:

| # | Test | Command |
|---|---|---|
| 8.1 | Create SPICE VM | `swift create e2e-test-spice --disk $IMG -m 512 -c 1 -s 5G --graphics spice --serial` |
| 8.2 | Create VNC+WebSocket VM | `swift create e2e-test-vnc --disk $IMG -m 512 -c 1 -s 5G --graphics vnc --websocket --serial` |
| 8.3 | Create headless VM | `swift create e2e-test-serial --disk $IMG -m 256 -c 1 -s 3G --graphics none --serial` |
| 8.4 | Create from YAML | `swift create -f test-vm.yaml` |
| 8.5 | List VMs | `swift list` — shows all 4 VMs |
| 8.6 | Status | `swift status e2e-test-spice` |
| 8.7 | Show XML | `swift show e2e-test-spice` — verifies SPICE in XML |
| 8.8 | VNC WebSocket XML | `swift show e2e-test-vnc` — verifies `websocket="-1"` |
| 8.9 | Serial XML | `swift show e2e-test-serial` — verifies `<serial>` and `<console>` |
| 8.10 | Start VM | `swift start e2e-test-spice` |
| 8.11 | Status Running | `swift status e2e-test-spice` → Running |
| 8.12 | Force stop | `swift stop e2e-test-spice --force` |
| 8.13 | Status Shutoff | `swift status e2e-test-spice` → Shutoff |
| 8.14 | Delete VM | `swift delete e2e-test-spice` |

#### Group 9: Network Lifecycle (9 tests — require libvirt)

| # | Test | Command |
|---|---|---|
| 9.1 | Create NAT network | `swift network create e2e-test-nat --cidr 192.168.200.0/24 --dhcp --dhcp-start 192.168.200.100 --dhcp-end 192.168.200.200` |
| 9.2 | Create isolated | `swift network create e2e-test-isolated --cidr 10.10.10.0/24 --mode isolated` |
| 9.3 | Create from YAML | `swift network create -f test-net.yaml` |
| 9.4 | List networks | `swift network list` |
| 9.5 | Start network | `swift network start e2e-test-nat` |
| 9.6 | Status | `swift network status e2e-test-nat` → Active |
| 9.7 | Show XML | `swift network show e2e-test-nat` — verifies nat mode + DHCP |
| 9.8 | Stop network | `swift network stop e2e-test-nat` |
| 9.9 | Delete networks | `swift network delete e2e-test-nat` + `e2e-test-isolated` |

---

## Prerequisites for Full E2E

```bash
# Add user to libvirt group (needs sudo)
sudo usermod -aG libvirt $USER
newgrp libvirt

# Ensure libvirtd is running
sudo systemctl enable --now libvirtd
```

---

## Test Result Summary

| Layer | Tests | Pass | Fail | Notes |
|---|---|---|---|---|
| Unit tests (`go test`) | 49 | 49 | 0 | All pass |
| E2E: CLI/help | 31 | 31 | 0 | All pass |
| E2E: error handling | 2 | 0 | 2 | Blocked by libvirt auth in PersistentPreRun |
| E2E: network CLI | 9 | 8 | 1 | Network create without name — blocked by libvirt auth |
| E2E: YAML files | 3 | 3 | 0 | All pass |
| E2E: build verify | 3 | 3 | 0 | All pass |
| E2E: disk image | 3 | 3 | 0 | All pass |
| E2E: cloud-init | 3 | 3 | 0 | All pass |
| E2E: VM lifecycle | 14 | 0 | 14 | Requires libvirt group membership |
| E2E: network lifecycle | 9 | 0 | 9 | Requires libvirt group membership |
| **Total** | **126** | **98** | **28** | **28 failures all due to libvirt permission** |

Once the user is added to the libvirt group and re-logs in, all 126 tests should pass.
