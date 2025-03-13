terraform {
  required_providers {
    vagrant = {
      source  = "bmatcuk/vagrant"
      version = "~> 3.0"
    }
  }
}

provider "vagrant" {}

resource "vagrant_machine" "almalinux" {
  provider    = "virtualbox"
  vagrantfile = "${path.module}/Vagrantfile"  # Ensure Vagrantfile is in the same directory
}

output "vm_ip" {
  value = "192.168.56.100"  # Adjust as needed
}

