terraform {
  required_providers {
    libvirt = {
      source  = "dmacvicar/libvirt"
      version = "~> 0.6.3"
    }
  }
}

provider "libvirt" {
  uri = "qemu:///system"
}

# Define the Cloud-Init disk with user data
resource "libvirt_cloudinit_disk" "cloudinit" {
  name = "cloudinit.iso"
  pool = "default"

  user_data = <<EOF
#cloud-config
hostname: almalinux-vm
manage_etc_hosts: true
users:
  - name: adamuser
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: users, admin
    shell: /bin/bash
    lock_passwd: false
    passwd: "password"   # 🔴 Consider replacing with a hashed password
ssh_pwauth: true
disable_root: false
chpasswd:
  expire: false
EOF
}

# Define the AlmaLinux disk volume
resource "libvirt_volume" "almalinux-disk" {
  name   = "almalinux-disk"
  pool   = "default"
  source = "/var/lib/libvirt/images/almalinux.qcow2"  # Path to AlmaLinux image
  format = "qcow2"
}

# Define the AlmaLinux virtual machine
resource "libvirt_domain" "almalinux-vm" {
  name   = "almalinux-vm"
  memory = 2048
  vcpu   = 2

  disk {
    volume_id = libvirt_volume.almalinux-disk.id
  }

  # Attach Cloud-Init disk
  cloudinit = libvirt_cloudinit_disk.cloudinit.id

  network_interface {
    network_name = "default"
    addresses    = ["192.168.122.100"]
  }

  console {
    type        = "pty"
    target_type = "serial"
    target_port = "0"
  }

  graphics {
    type        = "vnc"
    listen_type = "address"
  }
}

# Output the VM's assigned IP
output "vm_ip" {
  value = libvirt_domain.almalinux-vm.network_interface[0].addresses[0]
}

