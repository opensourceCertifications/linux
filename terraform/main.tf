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

resource "libvirt_volume" "almalinux-disk" {
  name   = "almalinux-disk"
  pool   = "default"
  source = "/var/lib/libvirt/images/almalinux.qcow2"  # Path to your AlmaLinux image
  format = "qcow2"
}

resource "libvirt_domain" "almalinux-vm" {
  name   = "almalinux-vm"
  memory = 2048
  vcpu   = 2

  disk {
    volume_id = libvirt_volume.almalinux-disk.id
  }

  network_interface {
    network_name = "default"
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

output "vm_ip" {
  value = libvirt_domain.almalinux-vm.network_interface[0].addresses[0]
}

