# linux
run this command before running ansible-playbook --check
if you just wanna run the playbook to get started then you can skip this step
the ansible should install the required collections for you
```bash
ansible-galaxy collection install -r requirements.yml
```


currently I'm seeing this
[vagrant@testenv testenv]$ go run test_chaos.go BreakBootLoader 192.168.56.10
2025/06/27 19:27:58 Running chaos function: BreakBootLoader
2025/06/27 19:27:58 No grub.cfg found!
2025/06/27 19:27:58 Function 'BreakBootLoader' returned error: No grub.cfg found!
exit status 1
[vagrant@testenv testenv]$ bash run_testenv.sh
2025/06/27 19:28:41 Waiting for monitor service to become available...
2025/06/27 19:28:41 Connected to monitor. Starting heartbeat...
2025/06/27 19:28:41 Chaos injector sleeping for 42.27327564s before next break...
2025/06/27 19:29:23 Failed to select a random break: no chaos functions registered
2025/06/27 19:29:23 Chaos injector sleeping for 48.140353757s before next break...
^Csignal: interrupt
[vagrant@testenv testenv]$


I'm guessing there's something wrong in my chaos script and how it handles the grub.cfg file
