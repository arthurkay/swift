# VM Operations

Swift manages virtual machine lifecycles through libvirt/KVM. All VM commands connect to the system libvirt daemon (`qemu:///system`).

## create

Creates and provisions a new virtual machine. Generates a QCOW2 disk image from a backing cloud image, creates a cloud-init seed ISO for automated OS configuration, and defines the VM in libvirt.

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--disk` | `-d` | (required) | Path to cloud image backing file |
| `--memory` | `-m` | 512 | Memory in MiB |
| `--cpu` | `-c` | 1 | Number of CPUs |
| `--storage` | `-s` | 8G | Disk size (e.g. 10G, 512M, 1T) |
| `--password` | `-p` | swift1234 | VM password |
| `--ssh-key` | | | SSH public key for the default user |
| `--user` | `-u` | swift | Cloud-init username |
| `--network` | `-n` | | Networks as `name:MODEL,...` |
| `--graphics` | `-g` | spice | Graphics type: spice, vnc, none |
| `--websocket` | | false | Enable WebSocket for browser console |
| `--serial` | | false | Add PTY serial console |
| `--file` | `-f` | | YAML configuration file |

### Examples

```bash
# Minimal create with defaults (512 MiB, 1 CPU, 8G disk, SPICE)
swift create myvm --disk ~/images/ubuntu-22.04.qcow2

# Custom resources
swift create myvm -d ~/images/ubuntu.qcow2 -m 2048 -c 4 -s 20G

# With SSH key
swift create myvm -d ~/images/fedora.qcow2 --ssh-key "ssh-rsa AAAA..."

# Multiple networks
swift create myvm -d ~/images/ubuntu.qcow2 --network default:e1000,mynet:virtio

# Headless with VNC + serial console
swift create myvm -d ~/images/ubuntu.qcow2 --graphics vnc --websocket --serial

# From simplified YAML config
swift create myvm -f vm.yaml

# Recreate from YAML round-trip
swift create -f <(swift show myvm -o yaml)
```

### What happens during create

1. **Cloud-init files generated** — `user-data` and `meta-data` in `/var/swift/<slug>/`
2. **QCOW2 disk created** — overlay image backed by your cloud image
3. **Seed ISO created** — `cidata.iso` for automated OS setup
4. **Domain defined in libvirt** — VM is registered but not running

### Defaults

- Memory: 512 MiB
- CPUs: 1
- Storage: 8 GiB
- Password: `swift1234`
- Username: `swift`
- Graphics: SPICE
- Disk format: QCOW2 overlay

## list

Lists all defined virtual machines with their UUID, ID, and current state.

```bash
swift list
swift vm      # alias
swift vms     # alias
```

Output:

```
NAME                 UUID                                   ID           STATE
----                 ----                                   --           -----
myvm                 0da02ace-0af8-428f-b6d8-adf0734be39e   4294967295   Shutoff
web-server           1b3c92df-3b4e-5678-9cde-bc8292fa4123   4294967295   Running
```

## start

Starts a defined virtual machine. If no name is provided, an interactive prompt lists available VMs.

```bash
swift start myvm
swift start          # interactive selection
```

## stop

Stops a running virtual machine. By default sends an ACPI shutdown signal for graceful stop. Use `--force` to immediately pull the plug.

```bash
swift stop myvm                # graceful ACPI shutdown
swift stop myvm --force        # immediate force stop
swift poweroff myvm            # alias for stop
```

When using graceful shutdown, Swift polls for up to 30 seconds. If the VM does not shut down in time, it automatically force-stops.

## delete

Undefines a virtual machine from libvirt and removes all associated files (disk images, cloud-init ISOs, configuration) from `/var/swift/<slug>/`.

```bash
swift delete myvm
swift delete          # interactive selection
```

Prompts for confirmation before removal.

## status

Shows the current state of a virtual machine.

```bash
swift status myvm
swift status          # interactive selection
```

States: `Running`, `Shutoff`, `Paused`, `Shutdown`, `Crashed`

## show

Displays the definition of a virtual machine. Defaults to YAML format; use `-o xml` for raw libvirt XML.

```bash
swift show myvm              # YAML (default)
swift show myvm -o yaml      # explicit YAML
swift show myvm -o xml       # raw libvirt XML
swift show                   # interactive selection
swift xml myvm               # alias for show
```

### YAML output format

The YAML output mirrors the libvirt XML structure exactly:

- XML attributes use `@` prefix (e.g. `@type`, `@unit`)
- Text content uses `_` key (e.g. `_: "524288"`)
- Multiple same-name children become YAML sequences

```yaml
domain:
    '@type': kvm
    name: myvm
    memory:
        '@unit': KiB
        _: "524288"
    vcpu:
        '@placement': static
        _: "1"
    devices:
        disk:
            '@device': disk
            '@type': file
            driver:
                '@name': qemu
                '@type': qcow2
            source:
                '@file': /var/swift/myvm/myvm.qcow2
            target:
                '@bus': virtio
                '@dev': vda
```

### Round-trip workflow

Save YAML, edit, and recreate:

```bash
# Export current definition
swift show myvm -o yaml > myvm.yaml

# Edit the YAML...
vim myvm.yaml

# Recreate from edited YAML
swift create -f myvm.yaml
```

Or in a single command:

```bash
swift create -f <(swift show myvm -o yaml)
```

## version

Prints the Swift version.

```bash
swift version
```

## Flags reference

### Common flags

| Flag | Description |
|------|-------------|
| `--help` | Show help for any command |
| `-o`, `--format` | Output format: `yaml` (default), `xml` (for show commands) |
| `-f`, `--file` | YAML configuration file (for create commands) |

### Interactive selection

All commands that accept a VM name (`start`, `stop`, `delete`, `status`, `show`) support interactive selection when no name is provided. A list of VMs with their current state is displayed for selection.
