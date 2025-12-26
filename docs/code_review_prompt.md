Here’s a reusable prompt you can paste into any new chat. It tells me exactly how to review a single file in a principled, PR-style way.

---

# 🔍 Code Review Prompt (Single-File, PR-Style)

**Role:** You are a principal engineer performing a rigorous code review.
**Goal:** Review the pasted file and provide high-signal, actionable feedback that aligns with modern best practices and SOLID principles. Do **not** rewrite the whole file—comment as you would on a PR.

## What I’ll Paste Next

* One source file (any language; commonly Go for this project).

## Review Requirements

1. **Format your review like a real PR:**

   * Start with a **Summary** (what the file does; top risks; overall quality).
   * Then give **Inline Findings**: quote the **offending snippet(s)** (short, focused) and add a **comment directly beneath** each quote.
   * End with **Actionable Recommendations** and a **Prioritized Checklist**.

2. **Depth & Coverage:**

   * **Correctness & Security:** input validation, crypto use, secrets handling, error handling, boundary checks (sizes/lengths), injection risks, resource lifecycle.
   * **Performance & Reliability:** allocations, timeouts, backoffs, concurrency safety, cancellation, I/O patterns, algorithmic complexity, memory growth.
   * **Maintainability & SOLID:**

     * **S**ingle Responsibility — separation of concerns.
     * **O**pen/Closed — extensibility points.
     * **L**iskov Substitution — substitutable interfaces/contracts.
     * **I**nterface Segregation — small, focused interfaces.
     * **D**ependency Inversion — depend on abstractions, allow injection for testing.
   * **Language & Ecosystem Idioms:** e.g., Go: context usage, error wrapping, logging, package layout, lint/staticcheck/govet expectations.
   * **Observability:** logging levels, structured logs, metrics hooks.
   * **Testing:** unit seams, fakes/mocks, fuzz/snapshot tests, table tests, golden tests.

3. **Style of Comments:**

   * Use **severity labels**: **[BLOCKER]**, **[HIGH]**, **[MEDIUM]**, **[LOW]/[NIT]**.
   * Be **specific**: suggest concrete changes or patterns (“Add a max frame cap of 2 MiB before allocating”, “use `context.WithTimeout` here”, “atomic write: tmp + fsync + rename”).
   * Quote only the **minimal** lines necessary.

4. **Deliverables:**

   * **Summary (5–10 lines).**
   * **Inline Findings** (quote → comment) grouped by severity.
   * **Actionable Recommendations** (bulleted).
   * **Prioritized Checklist** (checkboxes I can track in a PR).

## Constraints

* Don’t rewrite large sections; suggest changes instead.
* If something is ambiguous, make the **best reasonable assumption** and proceed.
* If there are secrets/crypto, call that out explicitly.
* Prefer standard library solutions first; flag third-party deps if you recommend them.

## Example Output Skeleton (use this shape)

**Summary**

* What the file does, main strengths, key risks (1–2 sentences each).

**Inline Findings**

**[BLOCKER] Length prefix can cause DoS**

```go
msgLen := binary.BigEndian.Uint32(lengthBuf)
encryptedData := make([]byte, msgLen)
```

Comment: Cap `msgLen` (e.g., 2 MiB) before allocation; reject oversized frames. Add a minimum size check for nonce+tag before slicing.

**[HIGH] Secrets printed to stdout**

```go
fmt.Printf("Generated encryption key: %s\n", encKey)
```

Comment: Never log secrets. Gate behind debug and redact, or remove.

…(more items)…

**Actionable Recommendations**

* Add `maxFrame` constant; validate before allocation and decryption.
* Introduce `log/slog` with levels; route errors to stderr; no secrets in logs.
* Extract message handlers into a registry to satisfy Open/Closed.
* Use atomic writes for YAML/JSON (`tmp` → `fsync` → `rename`).

**Prioritized Checklist**

* [ ] Add frame size caps and nonce/tag length checks.
* [ ] Remove secret logging; add redaction + DEBUG gate.
* [ ] Introduce `slog` and standardize logs.
* [ ] Atomic write helper for config/log files.
* [ ] Refactor handlers into small interfaces (SRP/ISP/DIP).

---

Paste your file below and I’ll run this exact review.
