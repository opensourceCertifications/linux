# linux certification
There's a problem with IT certificates, they're often very expensive and test the wrong thing. Take AWS for example, it's hard sure but it's all multiple choice, someone with perfect knowledge of the documentation and zero technical skills can, in theory, get any certification. The CKA is all practical sure but it's all browser based and limits you so much that it's really hard by restriction rather than actually testing the engineers ability. I aim to build a set of certifications that test what engineers do day to day.

* the certifications will be free to do locally thus lowering the barrier for entry
* they will be open source because tools generally are
* the user can bring any toy they want.

Currently I'm working on the Linux one, it has the following setup:
* 2 VMs a monitor and a testenv
the user will have access to the testenv but not the monitor. The monitor will have ssh access to the testenv and will login periodically to cause chaos.
* the user will get a list of stuff to do but will also be marked on if they manage to fix the chaos caused by the monitor.

curl -X POST http://192.168.56.1:$1 \
  -H "Content-Type: application/json" \
  -d '{"status": "operation_complete", "message": "done", "token": "$2"}'



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
