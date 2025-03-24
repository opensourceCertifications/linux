import random
import importlib
import os
import time
from agent.logger import log_action
from agent.sandbox import is_vm
import yaml

class ChaosEngine:
    def __init__(self):
        with open("config/rules.yaml") as f:
            self.rules = yaml.safe_load(f)
        self.memory = set()  # What we've sabotaged already

    def evaluate_system(self):
        # Placeholder: detect things like `ps`, installed packages, services
        return {
            "apt_installed": os.path.exists("/usr/bin/apt"),
            "sshd_running": os.system("systemctl is-active sshd >/dev/null") == 0,
        }

    def pick_sabotage(self, state):
        # Use weighted rules
        options = []
        for rule in self.rules:
            if not rule["condition"](state):
                continue
            if rule["id"] in self.memory:
                continue
            options.append((rule["id"], rule["weight"]))

        if not options:
            return None

        choice = random.choices([o[0] for o in options], weights=[o[1] for o in options])[0]
        self.memory.add(choice)
        return choice

    def execute(self, module_name):
        module = importlib.import_module(f"sabotage_modules.{module_name}")
        module.run()
        log_action(module_name)
