import time
import random
from agent.engine import ChaosEngine

def run_forever():
    engine = ChaosEngine()
    while True:
        delay = random.randint(180, 420)
        time.sleep(delay)
        state = engine.evaluate_system()
        sabotage = engine.pick_sabotage(state)
        if sabotage:
            engine.execute(sabotage)
