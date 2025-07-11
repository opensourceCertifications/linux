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
    monitor.vm.synced_folder "./monitor", "/home/vagrant/service/"
    shell_provision.call(monitor.vm)
  end

  config.vm.define "testenv" do |testenv|
    testenv.vm.box = "almalinux/9"
    testenv.vm.hostname = "testenv"
    testenv.vm.network "private_network", ip: "192.168.56.11"
    testenv.vm.synced_folder "./testenv/", "/home/vagrant/service/"
    shell_provision.call(testenv.vm)
  end
end
