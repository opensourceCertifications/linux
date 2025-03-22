Vagrant.configure("2") do |config|
  # Define Ubuntu VM
  config.vm.box = "ubuntu/jammy64"
  
  # Configure VM resources
  config.vm.provider "virtualbox" do |vb|
    vb.memory = "2048"
    vb.cpus = 2
  end

  # VM networking
  config.vm.network "forwarded_port", guest: 22, host: 2222

  # VM customization
  config.vm.provider "virtualbox" do |vb|
    vb.name = "ubuntu_vm"
    vb.memory = 2048
    vb.cpus = 2
  end

  config.vm.provision "shell", inline: <<-shell
    sudo apt update
    sudo apt upgrade -y
    sudo apt install -y git software-properties-common
    sudo add-apt-repository --yes --update ppa:ansible/ansible
    sudo apt install -y ansible
  shell

  # Provisioning (optional)
  # config.vm.provision "ansible" do |ansible|
  #   ansible.playbook = "setup_neovim.yml"
  # end
end
