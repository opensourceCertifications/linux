# frozen_string_literal: true

# ------------------------------------------------------------------------------
# Bootloader check lab — Vagrant topology
#
# Creates two AlmaLinux 9 VMs for local testing:
#   • monitor  — runs Ansible and orchestration (IP: 192.168.56.10)
#   • testenv  — target VM where boot files are corrupted/fixed (IP: 192.168.56.11)
#
# Provider: VirtualBox
#   - 2 vCPUs, 2GB RAM each (t3.small-ish)
#   - Paravirtualization: KVM (closer to Nitro/KVM behavior)
#   - NICs: virtio (lower overhead)
#   - CPU execution cap ~70% to mimic baseline throttling on dev laptops
#
# Sync:
#   - Host folder ./transfer  <->  guest /vagrant_transfer
#   - Used to pass the monitor’s SSH public key to testenv
#
# Provisioning flow:
#   1) monitor
#      - Generates an ed25519 keypair (vagrant user)
#      - Copies id_ed25519.pub into /vagrant_transfer/monitor.pub
#      - Runs ./monitor_script.sh as root (installs Go & Ansible)
#      - Exports TESTENV_ADDRESS and MONITOR_ADDRESS to /etc/environment
#   2) testenv
#      - dnf update/upgrade
#      - Waits until /vagrant_transfer/monitor.pub exists and is non-empty
#      - Appends it to ~vagrant/.ssh/authorized_keys (correct perms)
#
# Purpose:
#   - Deterministic local lab for Ansible-driven bootloader corruption checks.
#   - “monitor” controls “testenv” via SSH using the shared key.
#
# Usage:
#   - vagrant up
#   - vagrant ssh monitor   # run Ansible from /vagrant (your repo mount)
#   - vagrant ssh testenv   # inspect the target
#
# Tips:
#   - If you destroy/recreate testenv and see host key warnings from monitor:
#       ssh-keygen -R 192.168.56.11 && ssh-keyscan -H 192.168.56.11 >> ~/.ssh/known_hosts
#   - Default user is “vagrant”; synced folder mounted at /vagrant on each VM.
# ------------------------------------------------------------------------------

# Use Vagrant config version 2 (modern syntax)
Vagrant.configure('2') do |config|
  # -----------------------------
  # Shared values (used by both VMs)
  # -----------------------------
  monitor_ip = '192.168.56.10'     # host-only/private IP for the monitor VM
  testenv_ip = '192.168.56.11'     # host-only/private IP for the testenv VM
  shared_path = './transfer'       # local folder to exchange small artifacts (keys, etc.)
  host_mount  = '/vagrant_transfer' # mount point inside the VMs

  # Ensure the shared folder exists on the host before mounting
  FileUtils.mkdir_p(shared_path)

  # -----------------------------
  # Provider-level defaults for both VMs
  # -----------------------------
  %w[monitor testenv].each do |name|
    config.vm.define name do |m|
      m.vm.provider 'virtualbox' do |vb|
        vb.name = "yl-#{name}"      # nice, stable name in VirtualBox UI
        vb.memory = 2048            # t3.small-equivalent memory
        vb.cpus   = 2               # t3.small-equivalent vCPU count

        # Make the guest behave closer to KVM/Nitro for more realistic perf
        vb.customize ['modifyvm', :id, '--paravirtprovider', 'kvm']

        # Favor virtio NICs over e1000 for lower overhead
        vb.customize ['modifyvm', :id, '--nictype1', 'virtio']
        vb.customize ['modifyvm', :id, '--nictype2', 'virtio'] # (if a second NIC is used)

        # Soft limit CPU time to simulate baseline throttling on dev laptops
        # (0–100; 60–70 ~ “baseline with occasional bursts”)
        vb.customize ['modifyvm', :id, '--cpuexecutioncap', '70']
      end
    end
  end

  # -----------------------------
  # monitor VM
  # -----------------------------
  config.vm.define 'monitor' do |monitor|
    monitor.vm.box = 'almalinux/9'                      # base image
    monitor.vm.hostname = 'monitor'                     # /etc/hostname
    monitor.vm.network 'private_network', ip: monitor_ip # host-only network
    monitor.vm.synced_folder shared_path, host_mount # mount host ./transfer -> /vagrant_transfer

    # Create an SSH keypair (non-root) and drop the public key into the shared folder
    # so testenv can pick it up and authorize monitor access.
    monitor.vm.provision 'shell', privileged: false, inline: <<-SHELL
      ssh-keygen -t ed25519 -f $HOME/.ssh/id_ed25519 -N ""
      cp $HOME/.ssh/id_ed25519.pub #{host_mount}/monitor.pub
      echo "Host #{testenv_ip} testenv
        User root
        IdentityFile $HOME/.ssh/id_ed25519
        StrictHostKeyChecking accept-new
        UserKnownHostsFile $HOME/.ssh/known_hosts" > $HOME/.ssh/config
        chmod 600 $HOME/.ssh/config
        ssh-keyscan -H #{testenv_ip} >> "$HOME/.ssh/known_hosts" 2>/dev/null || true
        ssh-keyscan -H testenv        >> "$HOME/.ssh/known_hosts" 2>/dev/null || true
    SHELL

    # Run any additional bootstrap as root (external script you maintain)
    monitor.vm.provision 'shell', privileged: true, path: './scripts/monitor_script.sh'

    # Export IPs as environment variables system-wide (available after login)
    monitor.vm.provision 'shell', privileged: true, inline: <<-SHELL
      echo 'export TESTENV_ADDRESS=#{testenv_ip}' >> /etc/environment
      echo 'export MONITOR_ADDRESS=#{monitor_ip}' >> /etc/environment
      echo 'export ANSIBLE_VARS_PATH=/vagrant/monitor/ansible/ansible_vars.yml' >> /etc/environment
    SHELL
  end

  # -----------------------------
  # testenv VM
  # -----------------------------
  config.vm.define 'testenv' do |testenv|
    testenv.vm.box = 'almalinux/9' # base image
    testenv.vm.hostname = 'testenv'
    testenv.vm.network 'private_network', ip: testenv_ip
    testenv.vm.synced_folder shared_path, host_mount

    # Update/upgrade, then wait until monitor has written its public key
    # into the shared folder; once it exists, append to authorized_keys.
    testenv.vm.provision 'shell', privileged: true, inline: <<-SHELL
      sudo dnf update -y
      sudo dnf upgrade -y

      # Loop until the monitor's public key appears in the shared folder
      while true; do
        if [ "$(wc -l < #{host_mount}/monitor.pub)" -gt 0 ]; then
          # Append the key and fix permissions
          echo "Monitor public key found, installing..." >> /home/vagrant/ssh_key_install.loging
          cat #{host_mount}/monitor.pub >> $HOME/.ssh/authorized_keys
          break
        else
          echo "Waiting for monitor.pub to be written..." >> /home/vagrant/ssh_key_install.log
          sleep 2
        fi
      done
    SHELL
  end
end
