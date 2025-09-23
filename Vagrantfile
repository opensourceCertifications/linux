Vagrant.configure("2") do |config|
  # Define shared shell provisioning
  monitor_ip = "192.168.56.10"
  testenv_ip = "192.168.56.11"
  shared_path = "./transfer"
  host_mount  = "/vagrant_transfer"

  # Create the host folder if it doesn't exist
  Dir.mkdir(shared_path) unless Dir.exist?(shared_path)

  ["monitor","testenv"].each do |name|
    config.vm.define name do |m|
      m.vm.provider "virtualbox" do |vb|
        vb.name = "yl-#{name}"
        vb.memory = 2048                # t3.small RAM
        vb.cpus   = 2                   # t3.small vCPU count

        # Make VBox behave more like KVM/Nitro guests
        vb.customize ["modifyvm", :id, "--paravirtprovider", "kvm"]

        # Use virtio NIC (lower overhead than e1000)
        vb.customize ["modifyvm", :id, "--nictype1", "virtio"]
        vb.customize ["modifyvm", :id, "--nictype2", "virtio"] # if host-only is NIC2

        # Optional: soft cap overall CPU execution (mimics baseline throttle, not credits)
        # 60–70 feels like “baseline with occasional bursts” on dev laptops.
        vb.customize ["modifyvm", :id, "--cpuexecutioncap", "70"]
      end
    end
  end

  config.vm.define "monitor" do |monitor|
    monitor.vm.box = "almalinux/9"
    monitor.vm.hostname = "monitor"
    monitor.vm.network "private_network", ip: monitor_ip
    monitor.vm.synced_folder shared_path, host_mount
    monitor.vm.provision "shell", privileged: false, inline: <<-SHELL
      ssh-keygen -t ed25519 -f $HOME/.ssh/id_ed25519 -N ""
      cp $HOME/.ssh/id_ed25519.pub #{host_mount}/monitor.pub
    SHELL
    monitor.vm.provision "shell", privileged: true, path: "./monitor_script.sh"
    monitor.vm.provision "shell", privileged: true, inline: <<-SHELL
      echo 'export TESTENV_ADDRESS=#{testenv_ip}' >> /etc/environment
      echo 'export MONITOR_ADDRESS=#{monitor_ip}' >> /etc/environment
    SHELL
  end

  config.vm.define "testenv" do |testenv|
    testenv.vm.box = "almalinux/9"
    testenv.vm.hostname = "testenv"
    testenv.vm.network "private_network", ip: testenv_ip
    testenv.vm.synced_folder shared_path, host_mount
    testenv.vm.provision "shell", privileged: true, inline: <<-SHELL
      sudo dnf update -y
      sudo dnf upgrade -y
      while true; do
        if [ "$(wc -l < #{host_mount}/monitor.pub)" -gt 0 ]; then
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
