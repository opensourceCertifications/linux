.
├── ansible-terraform ## remove this directory and put the sub directories in the root
│   ├── ansible
│   │   └── setup-terraform.yaml
│   ├── main.tf ## create a terraform directory and put this file in it
## remove all the state files
│   ├── terraform.tfstate
│   ├── terraform.tfstate.1740663388.backup
│   ├── terraform.tfstate.backup
│   ├── terraform-vm
│   │   └──.terraform.lock.hcl
│   └── tfplan ## remove this file
├── breaks
│   ├── corrupt.sh
│   └── Dockerfile
└── README.md

# add in a gitignore file

