Vagrant.configure("2") do |config|
  # Define shared shell provisioning
  monitor_ip = "192.168.56.10"
  testenv_ip = "192.168.56.11"
  shared_path = "./transfer"
  host_mount  = "/vagrant_transfer"

  # Create the host folder if it doesn't exist
  Dir.mkdir(shared_path) unless Dir.exist?(shared_path)

  shell_provision = lambda do |vm|
    vm.provision "shell", privileged: true, inline: <<-SHELL
      # Install prerequisites for Homebrew and Go
      dnf update -y
      dnf install -y wget
      wget -P /tmp/ https://go.dev/dl/go1.24.5.linux-amd64.tar.gz
      tar -xvf /tmp/go1.24.5.linux-amd64.tar.gz -C /usr/local
      rm -f /tmp/go1.24.5.linux-amd64.tar.gz
      echo 'export GOROOT=/usr/local/go' >> /etc/environment
      bash -c 'echo "export PATH=\$PATH:/usr/local/go/bin" > /etc/profile.d/go.sh'
      chmod 644 /etc/profile.d/go.sh
    SHELL
  end

  config.vm.define "monitor" do |monitor|
    monitor.vm.box = "almalinux/9"
    monitor.vm.hostname = "monitor"
    monitor.vm.network "private_network", ip: monitor_ip
    monitor.vm.synced_folder shared_path, host_mount
    shell_provision.call(monitor.vm)
    monitor.vm.provision "shell", privileged: false, inline: <<-SHELL
      ssh-keygen -t ed25519 -f $HOME/.ssh/id_ed25519 -N ""
      cp $HOME/.ssh/id_ed25519.pub #{host_mount}/monitor.pub
    SHELL
    monitor.vm.provision "shell", privileged: true, inline: <<-SHELL
      echo 'export TESTENV_ADDRESS=#{testenv_ip}' >> /etc/environment
      echo 'export MONITOR_ADDRESS=#{monitor_ip}' >> /etc/environment
    SHELL
  end

  config.vm.define "testenv" do |testenv|
    testenv.vm.box = "almalinux/9"
    testenv.vm.hostname = "testenv"
    testenv.vm.network "private_network", ip: testenv_ip
    shell_provision.call(testenv.vm)
    testenv.vm.synced_folder shared_path, host_mount
    testenv.vm.provision "shell", privileged: false, inline: <<-SHELL
      # Wait for monitor.pub to appear
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
