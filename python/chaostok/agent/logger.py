import os
import datetime
import base64

LOG_PATH = "/var/lib/.chaostok/hiddenlog.log"

def log_action(action):
    os.makedirs(os.path.dirname(LOG_PATH), exist_ok=True)
    entry = f"{datetime.datetime.utcnow()} - {action}\n"
    with open(LOG_PATH, "a") as f:
        f.write(base64.b64encode(entry.encode()).decode() + "\n")
