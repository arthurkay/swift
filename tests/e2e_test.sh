#!/usr/bin/env bash
# Swift E2E Integration Tests
# Tests the full workflow from CLI to XML generation, cloud-init, disk images
# Requires: qemu-img, genisoimage/mkisofs, the swift binary
set -euo pipefail

SWIFT_BIN="./swift"
TEST_DIR="/tmp/swift-e2e-test-$$"
IMAGE="/tmp/swift-test-images/ubuntu-22.04-server-cloudimg-amd64.img"
PASSED=0
FAILED=0
ERRORS=""
export LIBVIRT_DEFAULT_URI="qemu:///system"

# Use sg to run swift with the libvirt group
run_swift() {
    local cmd="\"$TEST_DIR/swift\""
    for arg in "$@"; do
        cmd="$cmd \"$arg\""
    done
    sg libvirt -c "$cmd"
}

cleanup() {
    rm -rf "$TEST_DIR"
    # Clean up any test VMs/libvirt resources if possible
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh destroy e2e-test-spice" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh undefine e2e-test-spice --remove-all-storage" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh destroy e2e-test-vnc" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh undefine e2e-test-vnc --remove-all-storage" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh destroy e2e-test-serial" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh undefine e2e-test-serial --remove-all-storage" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh destroy e2e-test-yaml" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh undefine e2e-test-yaml --remove-all-storage" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-destroy e2e-test-nat" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-undefine e2e-test-nat" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-destroy e2e-test-isolated" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-undefine e2e-test-isolated" 2>/dev/null || true
}
trap cleanup EXIT

pass() {
    PASSED=$((PASSED + 1))
    echo "  ✓ $1"
}

fail() {
    FAILED=$((FAILED + 1))
    ERRORS="${ERRORS}\n  ✗ $1: $2"
    echo "  ✗ $1: $2"
}

setup() {
    mkdir -p "$TEST_DIR"
    cp "$SWIFT_BIN" "$TEST_DIR/swift"
    chmod +x "$TEST_DIR/swift"

    # Clean up any stale test resources from previous runs
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-destroy e2e-test-nat" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-destroy e2e-test-isolated" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-undefine e2e-test-nat" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-undefine e2e-test-isolated" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh destroy e2e-test-spice" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh destroy e2e-test-vnc" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh destroy e2e-test-serial" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh destroy e2e-test-yaml" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh undefine e2e-test-spice --remove-all-storage" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh undefine e2e-test-vnc --remove-all-storage" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh undefine e2e-test-serial --remove-all-storage" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh undefine e2e-test-yaml --remove-all-storage" 2>/dev/null || true
}

# ============================================================
# TEST GROUP 1: CLI Structure & Help
# ============================================================
test_cli_help() {
    echo ""
    echo "=== TEST GROUP 1: CLI Structure & Help ==="

    # 1.1 Version
    output=$(run_swift version 2>&1)
    if echo "$output" | grep -q "swift 0.4.0"; then
        pass "1.1 swift version prints correct version"
    else
        fail "1.1 swift version" "expected 'swift 0.4.0', got '$output'"
    fi

    # 1.2 Root help shows all commands
    output=$(run_swift --help 2>&1)
    for cmd in create list start stop delete status show network version; do
        if echo "$output" | grep -q "$cmd"; then
            pass "1.2 Root help includes '$cmd' command"
        else
            fail "1.2 Root help '$cmd'" "command not found in help output"
        fi
    done

    # 1.3 Create help shows all flags
    output=$(run_swift create --help 2>&1)
    for flag in disk memory cpu storage password ssh-key user network file graphics websocket serial; do
        if echo "$output" | grep -qF -- "-$flag"; then
            pass "1.3 Create help includes '--$flag' flag"
        else
            fail "1.3 Create help '--$flag'" "flag not found in create help"
        fi
    done

    # 1.4 Network subcommands
    output=$(run_swift network --help 2>&1)
    for sub in create list start stop delete status show; do
        if echo "$output" | grep -q "$sub"; then
            pass "1.4 Network help includes '$sub' subcommand"
        else
            fail "1.4 Network help '$sub'" "subcommand not found"
        fi
    done

    # 1.5 Stop has poweroff alias
    output=$(run_swift stop --help 2>&1)
    if echo "$output" | grep -qi "poweroff"; then
        pass "1.5 Stop command shows 'poweroff' alias"
    else
        fail "1.5 Stop alias" "poweroff alias not found"
    fi

    # 1.6 Show has xml alias
    output=$(run_swift show --help 2>&1)
    if echo "$output" | grep -qi "xml"; then
        pass "1.6 Show command shows 'xml' alias"
    else
        fail "1.6 Show alias" "xml alias not found"
    fi
}

# ============================================================
# TEST GROUP 2: Error Handling (no libvirt needed)
# ============================================================
test_error_handling() {
    echo ""
    echo "=== TEST GROUP 2: Error Handling ==="

    # 2.1 Create without --disk
    output=$(run_swift create test-vm 2>&1 || true)
    if echo "$output" | grep -qF -- "--disk is required"; then
        pass "2.1 Create without --disk shows error"
    else
        fail "2.1 Create without --disk" "expected '--disk is required' error, got: $output"
    fi

    # 2.2 Create without name or file
    output=$(run_swift create 2>&1 || true)
    if echo "$output" | grep -qi "requires\|required\|name"; then
        pass "2.2 Create without name shows error"
    else
        fail "2.2 Create without name" "expected error about name, got: $output"
    fi
}

# ============================================================
# TEST GROUP 3: Network XML Generation (via create --help shows flags)
# ============================================================
test_network_cli() {
    echo ""
    echo "=== TEST GROUP 3: Network CLI ==="

    # 3.1 Network create help
    output=$(run_swift network create --help 2>&1)
    for flag in cidr mode bridge device dhcp dhcp-start dhcp-end file; do
        if echo "$output" | grep -qF -- "-$flag"; then
            pass "3.1 Network create help includes '--$flag'"
        else
            fail "3.1 Network create help '--$flag'" "flag not found"
        fi
    done

    # 3.2 Network create without name (needs at least --cidr to get past validation)
    output=$(run_swift network create --cidr 192.168.1.0/24 2>&1 || true)
    if echo "$output" | grep -q "required"; then
        pass "3.2 Network create without name shows error"
    else
        fail "3.2 Network create without name" "expected error about name"
    fi
}

# ============================================================
# TEST GROUP 4: YAML Config Parsing (via Go tests, but verify file I/O)
# ============================================================
test_yaml_config() {
    echo ""
    echo "=== TEST GROUP 4: YAML Config Files ==="

    # 4.1 Create a VM YAML config
    cat > "$TEST_DIR/test-vm.yaml" << 'EOF'
name: e2e-test-yaml
disk: /tmp/swift-test-images/ubuntu-22.04-server-cloudimg-amd64.img
memory: 1024
cpu: 2
storage: 10G
username: testuser
password: testpass123
ssh_key: ssh-ed25519 AAAA-test-key
graphics: vnc
websocket: true
serial: true
networks:
  - name: default
    model: e1000
EOF
    if [ -f "$TEST_DIR/test-vm.yaml" ]; then
        pass "4.1 VM YAML config file created"
    else
        fail "4.1 VM YAML config" "file not created"
    fi

    # 4.2 Create a Network YAML config
    cat > "$TEST_DIR/test-net.yaml" << 'EOF'
name: e2e-test-nat
cidr: 192.168.100.0/24
mode: nat
dhcp: true
dhcp_start: 192.168.100.100
dhcp_end: 192.168.100.200
EOF
    if [ -f "$TEST_DIR/test-net.yaml" ]; then
        pass "4.2 Network YAML config file created"
    else
        fail "4.2 Network YAML config" "file not created"
    fi

    # 4.3 Create isolated network YAML
    cat > "$TEST_DIR/test-iso-net.yaml" << 'EOF'
name: e2e-test-isolated
cidr: 10.10.10.0/24
mode: isolated
dhcp: false
EOF
    if [ -f "$TEST_DIR/test-iso-net.yaml" ]; then
        pass "4.3 Isolated network YAML config created"
    else
        fail "4.3 Isolated network YAML" "file not created"
    fi
}

# ============================================================
# TEST GROUP 5: Swift Build Verification
# ============================================================
test_build_verification() {
    echo ""
    echo "=== TEST GROUP 5: Build Verification ==="

    # 5.1 Binary is executable
    if [ -x "$TEST_DIR/swift" ]; then
        pass "5.1 Swift binary is executable"
    else
        fail "5.1 Binary executable" "not executable"
    fi

    # 5.2 Binary is correct format
    output=$(file "$TEST_DIR/swift")
    if echo "$output" | grep -q "ELF"; then
        pass "5.2 Binary is valid ELF"
    else
        fail "5.2 Binary format" "not ELF: $output"
    fi

    # 5.3 Version matches go.mod
    output=$(run_swift version 2>&1)
    if echo "$output" | grep -q "0.4.0"; then
        pass "5.3 Version matches 0.4.0"
    else
        fail "5.3 Version" "expected 0.4.0"
    fi
}

# ============================================================
# TEST GROUP 6: Disk Image Creation (qemu-img)
# ============================================================
test_disk_creation() {
    echo ""
    echo "=== TEST GROUP 6: Disk Image Operations ==="

    VM_DIR="$TEST_DIR/disk-test"
    mkdir -p "$VM_DIR"

    # 6.1 Create a QCOW2 from backing image
    output=$(qemu-img create -f qcow2 -F qcow2 -b "$IMAGE" "$VM_DIR/test-disk.qcow2" 10737418240 2>&1)
    if [ -f "$VM_DIR/test-disk.qcow2" ]; then
        pass "6.1 QCOW2 disk image created from backing file"
    else
        fail "6.1 QCOW2 creation" "$output"
    fi

    # 6.2 Verify disk info
    output=$(qemu-img info "$VM_DIR/test-disk.qcow2" 2>&1)
    if echo "$output" | grep -q "qcow2"; then
        pass "6.2 Disk image is valid qcow2"
    else
        fail "6.2 Disk info" "not qcow2"
    fi
    if echo "$output" | grep -q "10 GiB"; then
        pass "6.2 Disk virtual size is 10G"
    else
        fail "6.2 Disk size" "expected 10G virtual"
    fi
}

# ============================================================
# TEST GROUP 7: Cloud-init ISO Creation
# ============================================================
test_cloudinit() {
    echo ""
    echo "=== TEST GROUP 7: Cloud-init ISO ==="

    CI_DIR="$TEST_DIR/cloudinit-test"
    mkdir -p "$CI_DIR"

    # 7.1 Create user-data
    cat > "$CI_DIR/user-data" << 'EOF'
#cloud-config
hostname: test-vm
manage_etc_hosts: true
users:
  - name: testuser
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: users, admin
    home: /home/testuser
    shell: /bin/bash
    lock_passwd: false
    ssh-authorized-keys:
      - ssh-ed25519 AAAA-test-key
ssh_pwauth: false
disable_root: false
chpasswd:
  list: |
     testuser:testpass123
  expire: False
package_update: true
packages:
  - qemu-guest-agent
final_message: "The system is finally up, after $UPTIME seconds"
EOF

    # 7.2 Create meta-data
    cat > "$CI_DIR/meta-data" << 'EOF'
instance-id: test-vm
local-hostname: test-vm
EOF

    # 7.3 Create ISO
    output=$(genisoimage -output "$CI_DIR/cidata.iso" -V cidata -r -J "$CI_DIR/user-data" "$CI_DIR/meta-data" 2>&1)
    if [ -f "$CI_DIR/cidata.iso" ]; then
        pass "7.1 Cloud-init seed ISO created"
    else
        fail "7.1 ISO creation" "$output"
    fi

    # 7.4 Verify ISO is valid
    output=$(file "$CI_DIR/cidata.iso")
    if echo "$output" | grep -qi "iso\|ISO"; then
        pass "7.2 ISO file is valid ISO format"
    else
        fail "7.2 ISO format" "$output"
    fi

    # 7.5 Verify ISO size > 0
    size=$(stat -c%s "$CI_DIR/cidata.iso")
    if [ "$size" -gt 0 ]; then
        pass "7.3 ISO file size is ${size} bytes (non-zero)"
    else
        fail "7.3 ISO size" "zero bytes"
    fi
}

# ============================================================
# TEST GROUP 8: VM Lifecycle (requires libvirt)
# ============================================================
test_vm_lifecycle() {
    echo ""
    echo "=== TEST GROUP 8: VM Lifecycle (libvirt) ==="

    if ! sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh list --all" >/dev/null 2>&1; then
        echo "  ⚠ Skipping: libvirt not accessible (need libvirt group)"
        return
    fi

    # Ensure the default network exists (needed by VMs without explicit --network)
    if ! sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-info default" >/dev/null 2>&1; then
        sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-define /usr/share/libvirt/networks/default.xml" 2>/dev/null || true
        sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-start default" 2>/dev/null || true
    fi

    # 8.1 Create VM with SPICE
    output=$(run_swift create e2e-test-spice \
        --disk "$IMAGE" \
        --memory 512 \
        --cpu 1 \
        --storage 5G \
        --graphics spice \
        --serial 2>&1 || true)
    if echo "$output" | grep -q "created successfully"; then
        pass "8.1 Create VM with SPICE graphics"
    else
        fail "8.1 Create SPICE VM" "$output"
    fi

    # 8.2 Create VM with VNC + WebSocket
    output=$(run_swift create e2e-test-vnc \
        --disk "$IMAGE" \
        --memory 512 \
        --cpu 1 \
        --storage 5G \
        --graphics vnc \
        --websocket \
        --serial 2>&1 || true)
    if echo "$output" | grep -q "created successfully"; then
        pass "8.2 Create VM with VNC + WebSocket"
    else
        fail "8.2 Create VNC VM" "$output"
    fi

    # 8.3 Create VM with no graphics
    output=$(run_swift create e2e-test-serial \
        --disk "$IMAGE" \
        --memory 256 \
        --cpu 1 \
        --storage 3G \
        --graphics none \
        --serial 2>&1 || true)
    if echo "$output" | grep -q "created successfully"; then
        pass "8.3 Create VM with no graphics (headless)"
    else
        fail "8.3 Create headless VM" "$output"
    fi

    # 8.4 Create VM from YAML
    output=$(run_swift create -f "$TEST_DIR/test-vm.yaml" 2>&1 || true)
    if echo "$output" | grep -q "created successfully"; then
        pass "8.4 Create VM from YAML config"
    else
        fail "8.4 Create from YAML" "$output"
    fi

    # 8.5 List VMs
    output=$(run_swift list 2>&1 || true)
    for vm in e2e-test-spice e2e-test-vnc e2e-test-serial e2e-test-yaml; do
        if echo "$output" | grep -q "$vm"; then
            pass "8.5 List shows '$vm'"
        else
            fail "8.5 List '$vm'" "VM not found in list"
        fi
    done

    # 8.6 Status
    output=$(run_swift status e2e-test-spice 2>&1 || true)
    if echo "$output" | grep -q "e2e-test-spice"; then
        pass "8.6 Status shows VM state"
    else
        fail "8.6 Status" "$output"
    fi

    # 8.7 Show XML
    output=$(run_swift show e2e-test-spice -o xml 2>&1 || true)
    if echo "$output" | grep -q "e2e-test-spice"; then
        pass "8.7 Show displays XML definition"
    else
        fail "8.7 Show XML" "XML not displayed"
    fi
    if echo "$output" | grep -qE "type=['\"]spice['\"]"; then
        pass "8.7 Show XML contains SPICE graphics"
    else
        fail "8.7 Show SPICE" "SPICE graphics not in XML"
    fi

    # 8.8 Show VNC VM XML has websocket
    output=$(run_swift show e2e-test-vnc -o xml 2>&1 || true)
    if echo "$output" | grep -qE "websocket=['\"]-1['\"]"; then
        pass "8.8 VNC VM XML contains websocket attribute"
    else
        fail "8.8 VNC WebSocket" "websocket not in XML"
    fi

    # 8.9 Show serial VM XML has serial/console
    output=$(run_swift show e2e-test-serial -o xml 2>&1 || true)
    if echo "$output" | grep -q '<serial'; then
        pass "8.9 Headless VM XML contains serial element"
    else
        fail "8.9 Serial element" "serial not in XML"
    fi
    if echo "$output" | grep -q '<console'; then
        pass "8.9 Headless VM XML contains console element"
    else
        fail "8.9 Console element" "console not in XML"
    fi

    # 8.9a Show default output is YAML
    output=$(run_swift show e2e-test-spice 2>&1 || true)
    if echo "$output" | grep -q "domain:"; then
        pass "8.9a Show default output is YAML format"
    else
        fail "8.9a Show YAML" "expected 'domain:' key in YAML output"
    fi
    if echo "$output" | grep -q "'@type':"; then
        pass "8.9a Show YAML contains @-prefixed attributes"
    else
        fail "8.9a Show YAML attributes" "expected '@type' attribute"
    fi

    # 8.9b YAML round-trip: save YAML, recreate from it
    # First save the YAML output from a VM that exists
    run_swift show e2e-test-vnc -o yaml > "$TEST_DIR/roundtrip.yaml" 2>/dev/null
    if [ -f "$TEST_DIR/roundtrip.yaml" ] && grep -q "domain:" "$TEST_DIR/roundtrip.yaml"; then
        # Undefine the VM so we can recreate from YAML (use virsh directly, skip interactive prompt)
        sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh undefine e2e-test-vnc --remove-all-storage" 2>/dev/null || true
        # Recreate from YAML
        output=$(run_swift create -f "$TEST_DIR/roundtrip.yaml" 2>&1 || true)
        if echo "$output" | grep -q "VM defined from YAML definition"; then
            pass "8.9b YAML round-trip: VM recreated from YAML"
        else
            fail "8.9b YAML round-trip" "expected 'VM defined from YAML definition', got: $output"
        fi
    else
        fail "8.9b YAML round-trip" "YAML file not created or invalid"
    fi

    # 8.10 Start VM
    output=$(run_swift start e2e-test-spice 2>&1 || true)
    if echo "$output" | grep -q "started"; then
        pass "8.10 Start VM"
    else
        fail "8.10 Start VM" "$output"
    fi

    # 8.11 Status after start
    output=$(run_swift status e2e-test-spice 2>&1 || true)
    if echo "$output" | grep -q "Running"; then
        pass "8.11 Status shows Running after start"
    else
        fail "8.11 Status Running" "$output"
    fi

    # 8.12 Stop VM (force)
    output=$(run_swift stop e2e-test-spice --force 2>&1 || true)
    if echo "$output" | grep -q "force stopped"; then
        pass "8.12 Force stop VM"
    else
        fail "8.12 Force stop" "$output"
    fi

    # 8.13 Status after stop
    output=$(run_swift status e2e-test-spice 2>&1 || true)
    if echo "$output" | grep -q "Shutoff"; then
        pass "8.13 Status shows Shutoff after stop"
    else
        fail "8.13 Status Shutoff" "$output"
    fi

    # 8.14 Delete VM (with confirmation)
    output=$(echo "y" | run_swift delete e2e-test-spice 2>&1 || true)
    if echo "$output" | grep -q "deleted successfully"; then
        pass "8.14 Delete VM"
    else
        fail "8.14 Delete VM" "$output"
    fi
}

# ============================================================
# TEST GROUP 9: Network Lifecycle (requires libvirt)
# ============================================================
test_network_lifecycle() {
    echo ""
    echo "=== TEST GROUP 9: Network Lifecycle (libvirt) ==="

    if ! sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh list --all" >/dev/null 2>&1; then
        echo "  ⚠ Skipping: libvirt not accessible (need libvirt group)"
        return
    fi

    # 9.1 Create NAT network
    output=$(run_swift network create e2e-test-nat \
        --cidr 192.168.200.0/24 \
        --dhcp \
        --dhcp-start 192.168.200.100 \
        --dhcp-end 192.168.200.200 2>&1 || true)
    if echo "$output" | grep -q "created successfully"; then
        pass "9.1 Create NAT network with DHCP"
    else
        fail "9.1 Create NAT network" "$output"
    fi

    # 9.2 Create isolated network
    output=$(run_swift network create e2e-test-isolated \
        --cidr 10.10.10.0/24 \
        --mode isolated 2>&1 || true)
    if echo "$output" | grep -q "created successfully"; then
        pass "9.2 Create isolated network"
    else
        fail "9.2 Create isolated network" "$output"
    fi

    # 9.3 Create from YAML (undefine first since 9.1 already created e2e-test-nat)
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-destroy e2e-test-nat" 2>/dev/null || true
    sg libvirt -c "LIBVIRT_DEFAULT_URI=qemu:///system virsh net-undefine e2e-test-nat" 2>/dev/null || true
    output=$(run_swift network create -f "$TEST_DIR/test-net.yaml" 2>&1 || true)
    if echo "$output" | grep -q "created successfully"; then
        pass "9.3 Create network from YAML"
    else
        fail "9.3 Create from YAML" "$output"
    fi

    # 9.4 List networks
    output=$(run_swift network list 2>&1 || true)
    for net in e2e-test-nat e2e-test-isolated; do
        if echo "$output" | grep -q "$net"; then
            pass "9.4 Network list shows '$net'"
        else
            fail "9.4 Network list '$net'" "not found"
        fi
    done

    # 9.5 Start network
    output=$(run_swift network start e2e-test-nat 2>&1 || true)
    if echo "$output" | grep -q "started"; then
        pass "9.5 Start network"
    else
        fail "9.5 Start network" "$output"
    fi

    # 9.6 Status network
    output=$(run_swift network status e2e-test-nat 2>&1 || true)
    if echo "$output" | grep -q "Active"; then
        pass "9.6 Network status shows Active"
    else
        fail "9.6 Network status" "$output"
    fi

    # 9.7 Show network XML
    output=$(run_swift network show e2e-test-nat -o xml 2>&1 || true)
    if echo "$output" | grep -qE "mode=['\"]nat['\"]"; then
        pass "9.7 Network XML shows nat mode"
    else
        fail "9.7 Network XML" "nat mode not found"
    fi
    if echo "$output" | grep -q 'dhcp'; then
        pass "9.7 Network XML shows DHCP config"
    else
        fail "9.7 Network DHCP" "dhcp not found in XML"
    fi

    # 9.7a Show network default output is YAML
    output=$(run_swift network show e2e-test-nat 2>&1 || true)
    if echo "$output" | grep -q "network:"; then
        pass "9.7a Network show default output is YAML format"
    else
        fail "9.7a Network show YAML" "expected 'network:' key in YAML output"
    fi
    if echo "$output" | grep -q "'@mode': nat"; then
        pass "9.7a Network show YAML contains nat mode"
    else
        fail "9.7a Network show YAML mode" "expected '@mode: nat'"
    fi

    # 9.8 Stop network
    output=$(run_swift network stop e2e-test-nat 2>&1 || true)
    if echo "$output" | grep -q "stopped"; then
        pass "9.8 Stop network"
    else
        fail "9.8 Stop network" "$output"
    fi

    # 9.9 Delete networks
    output=$(run_swift network delete e2e-test-nat 2>&1 || true)
    if echo "$output" | grep -q "deleted successfully"; then
        pass "9.9 Delete NAT network"
    else
        fail "9.9 Delete NAT" "$output"
    fi

    output=$(run_swift network delete e2e-test-isolated 2>&1 || true)
    if echo "$output" | grep -q "deleted successfully"; then
        pass "9.9 Delete isolated network"
    else
        fail "9.9 Delete isolated" "$output"
    fi
}

# ============================================================
# MAIN
# ============================================================
echo "============================================="
echo "  Swift E2E Integration Tests"
echo "============================================="

setup

test_cli_help
test_error_handling
test_network_cli
test_yaml_config
test_build_verification
test_disk_creation
test_cloudinit
test_vm_lifecycle
test_network_lifecycle

echo ""
echo "============================================="
echo "  Results: $PASSED passed, $FAILED failed"
echo "============================================="

if [ $FAILED -gt 0 ]; then
    echo -e "\nFailures:$ERRORS"
    exit 1
fi

echo ""
echo "All tests passed!"
exit 0
