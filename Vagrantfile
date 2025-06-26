Vagrant.configure("2") do |config|
  # Define shared shell provisioning
  shell_provision = lambda do |vm|
    vm.provision "shell", inline: <<-SHELL
      sudo systemctl enable --now sshd
      sudo dnf update -y
      sudo dnf install -y git
      su - vagrant -c 'NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"'
      echo 'export PATH=$PATH:/home/linuxbrew/.linuxbrew/bin' >> /home/vagrant/.bashrc
      echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> /home/vagrant/.bashrc
      echo "brew install golang" >> /home/vagrant/first_run.sh
      su - vagrant -c 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)" && brew install go'
      #cd /home/vagrant
      #su - vagrant -c "cd /home/vagrant && go mod init monitor && go mod tidy"
      #cd /usr/bin/
      #su - root -c "cd /usr/bin && go mod init monitor && go mod tidy"
      #sudo systemctl daemon-reexec
      #sudo systemctl daemon-reload
    SHELL
  end

  config.vm.define "monitor" do |monitor|
    monitor.vm.box = "almalinux/9"
    monitor.vm.hostname = "monitor"
    monitor.vm.network "private_network", ip: "192.168.56.10"
    #monitor.vm.provision "file", source: "monitor_service.go", destination: "/home/vagrant/monitor_service.go"
    monitor.vm.synced_folder "./monitor", "/home/vagrant/service/"
#    monitor.vm.provision "file", source: "monitor.service", destination: "/tmp/monitor.service"
    shell_provision.call(monitor.vm)
#    monitor.vm.provision "shell", inline: <<-SHELL
#      sudo cp /tmp/monitor_service.go /usr/bin/monitor_service.go
#      sudo cp /tmp/monitor.service /etc/systemd/system/monitor.service
#      systemctl enable monitor.service
#      systemctl start monitor.service
#    SHELL
  end

  config.vm.define "testenv" do |testenv|
    testenv.vm.box = "almalinux/9"
    testenv.vm.hostname = "testenv"
    testenv.vm.network "private_network", ip: "192.168.56.11"
#    testenv.vm.provision "file", source: "test_environment.go", destination: "/home/vagrant/test_environment.go"
    testenv.vm.synced_folder "./testenv/", "/home/vagrant/service/"
#    testenv.vm.provision "file", source: "testenv.service", destination: "/tmp/testenv.service"
    shell_provision.call(testenv.vm)
#    testenv.vm.provision "shell", inline: <<-SHELL
#      sudo cp /tmp/test_environment.go /usr/bin/test_environment.go
#      sudo cp /tmp/testenv.service /etc/systemd/system/testenv.service
#      systemctl enable testenv.service
#      systemctl start testenv.service
#    SHELL
  end
end
