# Chaos Execution and Coordination Architecture

## Overview

This document describes the architecture and design for a secure, agentless chaos execution framework as part of an open-source, high-difficulty Linux certification system. The system is designed to:

- Execute chaos functions on a remote test environment (`testenv`)
- Log all chaos execution data on a control node (`monitor`)
- Prevent test candidates from tampering with or detecting the chaos process
- Operate using Go binaries without requiring any long-lived agents or services on `testenv`

---

## System Components

### 1. Monitor (Control Node)

- Hosts the main application responsible for initiating chaos.
- Selects and compiles chaos Go scripts on-demand.
- Listens on an ephemeral TCP port for responses from executed chaos binaries.
- Transfers compiled binaries to `testenv` via SSH.
- Executes chaos scripts remotely using SSH.
- Receives structured logs or completion signals via TCP.

### 2. Testenv (Target Node)

- Receives compiled Go binaries from `monitor`.
- Executes binaries in isolated, short-lived processes.
- Connects back to `monitor` on the assigned port to deliver logs or status.
- Has no long-running daemons or persistent chaos code.

### 3. Chaos Module (Go Script)

- Compiled Go code that implements a specific chaos function.
- Receives monitor IP, port, and token as arguments or environment variables.
- Connects to the monitor, authenticates with a one-time token, executes the chaos action, then sends logs/results.

---

## Chaos Execution Workflow

1. **Monitor Preparation**
   - Random high port selected.
   - One-time token generated.
   - TCP listener started on selected port.

2. **Script Compilation**
   - Chaos script is compiled using `go build` with optional parameters passed as environment variables or CLI flags.
   - Compilation is fast (<1s) and creates a static binary (~2–5MB).

3. **Deployment**
   - Binary is copied to `testenv` via SCP or SSH file transfer.

4. **Execution**
   - Binary is executed remotely over SSH.
   - It performs the chaos function and connects back to `monitor` with the token.

5. **Log Collection**
   - Monitor receives and verifies token.
   - Logs, results, or error messages are received and stored.
   - Port is closed after response is received or after timeout.

6. **Cleanup**
   - Monitor sends command to delete the binary on `testenv`.
   - Listener is closed and resources are released.

---

## Security Considerations

- **One-Time Token**: Prevents unauthorized or spoofed responses.
- **No Agents**: `testenv` runs no persistent processes that can be manipulated by the user.
- **SSH Transport**: All commands and transfers use encrypted SSH channels.
- **Ephemeral Ports**: Only open for brief periods, reducing surface area.
- **Audit Logging**: `monitor` stores all chaos activity for grading and verification.

---

## Extensibility

- New chaos functions can be added as Go scripts with a standard function signature.
- Scripts can be compiled dynamically and invoked via CLI or API.
- Future integration with web interfaces or distributed systems is possible.

---

## Future Improvements

- Add support for gRPC or HTTP control plane.
- Encrypt and compress TCP messages for performance and confidentiality.
- Use digital signatures to further verify authenticity of chaos responses.
- Build a plugin system or DSL to define chaos workflows declaratively.

---

## Summary

This architecture enables a secure, agentless, and auditable chaos execution system for use in high-assurance Linux certifications. It balances control, simplicity, and flexibility using Go's strengths in networking, portability, and compilation speed.
