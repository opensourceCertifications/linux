def condition(state):
    return state.get("apt_installed", False)

def run():
    with open("/etc/apt/sources.list", "a") as f:
        f.write("# added by chaos\n")
        f.write("deb http://not.a.real.server/debian stable main\n")
