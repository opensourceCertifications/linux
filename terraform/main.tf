terraform {
  required_providers {
    vagrant = {
      source  = "bmatcuk/vagrant"
      version = "~> 3.0"
    }
  }
}

provider "vagrant" {
  cwd = "/home/adamuser/linux/vagrant-almalinux/"  # Path to Vagrantfile
}

resource "vagrant_vm" "almalinux-vm" {
  name   = "almalinux-vm"
  box    = "generic/almalinux9"
  memory = 2048
  cpus   = 2

  network {
    type = "private_network"
    ip   = "192.168.56.100"  # Adjust if necessary
  }

  provisioner "remote-exec" {
    inline = [
      "sudo dnf install -y cloud-utils",
      "sudo systemctl enable --now sshd"
    ]

    connection {
      type     = "ssh"
      user     = "vagrant"
      password = "vagrant"
      host     = self.network[0].ip
    }
  }
}

output "vm_ip" {
  value = vagrant_vm.almalinux-vm.network[0].ip
}

