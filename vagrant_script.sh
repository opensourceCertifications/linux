dnf update -y
dnf install -y wget
wget -P /tmp/ https://go.dev/dl/go1.24.5.linux-amd64.tar.gz
tar -xvf /tmp/go1.24.5.linux-amd64.tar.gz -C /usr/local
rm -f /tmp/go1.24.5.linux-amd64.tar.gz
echo 'export GOROOT=/usr/local/go' >> /etc/environment
bash -c 'echo "export PATH=\$PATH:/usr/local/go/bin" > /etc/profile.d/go.sh'
chmod 644 /etc/profile.d/go.sh
dnf install -y python3-pip
su vagrant -c "python3 -m pip install --user ansible"
