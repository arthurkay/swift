# YAML Round-Trip Workflow

Swift supports lossless YAML round-tripping: export a VM or network definition as YAML, edit it, and recreate from it. The YAML mirrors the libvirt XML structure exactly.

## VM round-trip

### Export

```bash
# Default YAML format
swift show myvm > myvm.yaml

# Explicit YAML
swift show myvm -o yaml > myvm.yaml

# Raw XML (for comparison)
swift show myvm -o xml > myvm.xml
```

### Edit

The YAML uses `@` prefix for XML attributes and `_` for text content:

```yaml
domain:
    '@type': kvm
    name: myvm
    memory:
        '@unit': KiB
        _: "524288"
```

Common edits:

```yaml
# Change memory from 512 MiB to 2048 MiB
    memory:
        '@unit': KiB
        _: "2097152"       # 2048 * 1024

# Change CPUs from 1 to 4
    vcpu:
        '@placement': static
        _: "4"
```

### Recreate

```bash
# From file
swift create -f myvm.yaml

# One-liner (no file needed)
swift create -f <(swift show myvm -o yaml)
```

### Full example

```bash
# 1. Export
swift show web-server -o yaml > web-server.yaml

# 2. Edit memory and CPU
sed -i 's/_: "524288"/_: "2097152"/' web-server.yaml
sed -i 's/_: "1"/_: "4"/' web-server.yaml

# 3. Undefine old VM (preserves disk)
virsh undefine web-server

# 4. Recreate with new config
swift create -f web-server.yaml

# 5. Verify
swift show web-server -o yaml | grep -A1 memory
```

## Network round-trip

### Export

```bash
swift network show mynet > mynet.yaml
swift network show mynet -o yaml > mynet.yaml
```

### Edit

```yaml
network:
    name: mynet
    forward:
        '@mode': nat
    bridge:
        '@name': virbr1
    ip:
        '@address': 192.168.100.1    # change subnet
        '@netmask': 255.255.255.0
```

### Recreate

```bash
# Undefine first
virsh net-undefine mynet

# Recreate
swift network create -f mynet.yaml
```

### Full example

```bash
# 1. Export
swift network show prod-net -o yaml > prod-net.yaml

# 2. Change subnet from 192.168.100.0/24 to 10.0.0.0/24
sed -i 's/192.168.100.1/10.0.0.1/' prod-net.yaml
sed -i 's/255.255.255.0/255.255.255.0/' prod-net.yaml

# 3. Stop and undefine
swift network stop prod-net
virsh net-undefine prod-net

# 4. Recreate
swift network create -f prod-net.yaml

# 5. Start
swift network start prod-net
```

## Combining simplified and XML-mirroring formats

You can mix formats within a workflow:

```bash
# Start from simplified YAML
cat > base.yaml << 'EOF'
name: myvm
disk: /var/lib/libvirt/images/ubuntu-22.04.qcow2
memory: 2048
cpu: 2
graphics: vnc
EOF

# Create the VM
swift create myvm -f base.yaml

# Export the full definition for fine-tuning
swift show myvm -o yaml > full.yaml

# Edit specific XML attributes (e.g. add QEMU args, change machine type)
vim full.yaml

# Recreate with full definition
virsh undefine myvm --remove-all-storage
swift create -f full.yaml
```

## Validation

Swift performs these checks during the round-trip:

1. **Format detection** — Verifies the YAML has a `domain` or `network` top-level key with a map value
2. **YAML parsing** — Validates YAML syntax
3. **XML generation** — Converts YAML to valid libvirt XML
4. **File existence** — Warns if referenced disk/ISO files do not exist
5. **Libvirt validation** — Libvirt validates the XML before defining

Errors are reported at the first failing step:

```
Error: convert YAML to XML: ...          # YAML structure issue
Warning: referenced file does not exist  # file missing (warning only)
Error: define domain: virError(...)      # libvirt rejected the XML
```

## Limitations

- **Element ordering** — Go maps iterate in random order, so YAML output is alphabetically sorted. The reconstituted XML has different element ordering than the original. This is functionally correct — libvirt does not care about element order.
- **XML processing instructions** — The `<?xml version="1.0"?>` header is discarded during conversion. Libvirt regenerates it.
- **No provisioning** — The XML-mirroring path defines the domain in libvirt but does not create disk images, cloud-init ISOs, or project directories.
- **Comments lost** — XML comments are not preserved in the YAML output.
