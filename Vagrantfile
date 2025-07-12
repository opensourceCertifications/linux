Vagrant.configure("2") do |config|
  # Define shared shell provisioning
  shell_provision = lambda do |vm|
    vm.provision "shell", inline: <<-SHELL
      # Install prerequisites for Homebrew and Go
      sudo dnf update -y
      sudo dnf install -y git curl gcc make procps-ng glibc-devel

      # Enable SSHD
      sudo systemctl enable --now sshd

      # Install Homebrew as vagrant user
      su - vagrant -c 'NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"'

      # Set up Homebrew environment in .bashrc
      echo 'export PATH=/home/linuxbrew/.linuxbrew/bin:/home/linuxbrew/.linuxbrew/sbin:$PATH' >> /home/vagrant/.bashrc
      echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> /home/vagrant/.bashrc

      # Install Go with explicit glibc handling
      su - vagrant -c '
        export PATH=/home/linuxbrew/.linuxbrew/bin:/home/linuxbrew/.linuxbrew/sbin:$PATH
        eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
        brew install glibc
        brew install go
      '
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