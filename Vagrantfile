Vagrant.configure("2") do |config|
  monitor_ip = "192.168.56.10"
  testenv_ip = "192.168.56.11"
  monitor_port = "9000"
  # Define shared shell provisioning
  shell_provision = lambda do |vm|
    vm.provision "shell", inline: <<-SHELL
      # Install prerequisites for Homebrew and Go
      dnf update -y
      dnf install -y git wget

      # Enable SSHD
      systemctl enable --now sshd
      wget -P /tmp/ https://go.dev/dl/go1.24.5.linux-amd64.tar.gz
      tar -xvf /tmp/go1.24.5.linux-amd64.tar.gz -C /usr/local
      rm -f /tmp/go1.24.5.linux-amd64.tar.gz
      echo 'export MONITOR_PORT=#{monitor_port}' >> /etc/environment
      echo 'export GOROOT=/usr/local/go' >> /etc/environment
      bash -c 'echo "export PATH=\$PATH:/usr/local/go/bin" > /etc/profile.d/go.sh'
      chmod 644 /etc/profile.d/go.sh
    SHELL
  end

  config.vm.define "monitor" do |monitor|
    monitor.vm.box = "almalinux/9"
    monitor.vm.hostname = "monitor"
    monitor.vm.network "private_network", ip: monitor_ip
    monitor.vm.synced_folder "./monitor", "/home/vagrant/service/"
    shell_provision.call(monitor.vm)
  end

  config.vm.define "testenv" do |testenv|
    testenv.vm.box = "almalinux/9"
    testenv.vm.hostname = "testenv"
    testenv.vm.network "private_network", ip: testenv_ip
    testenv.vm.synced_folder "./testenv/", "/home/vagrant/service/"
    shell_provision.call(testenv.vm)
    testenv.vm.provision "shell", inline: <<-SHELL
      echo 'export MONITOR_ADDRESS=#{monitor_ip}' >> /etc/environment
    SHELL
  end
end
