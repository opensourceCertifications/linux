# Troubleshooting

    ## Go modules & linter scope
    - If `golangci-lint run` shows _“pattern ./...: directory prefix . does not contain modules …”_, run from within `monitor/go`, or pass paths explicitly:  
      `golangci-lint run monitor/go/...`

    ## AES‑GCM decryption failures
    - Ensure the monitor and agent use **the same hex key** (must decode to **32 bytes**).  
    - The wire format is: `uint32 big‑endian length` + `nonce(12)` + `ciphertext`.  
    - **Algorithm**: AES‑GCM (Go `crypto/cipher` with `cipher.NewGCM`).
- **Nonce**: 12 bytes (generated per message).
- **Key**: hex decoded; required length 32 bytes for AES‑256.

    ## SSH EOF during remote exec
    - Closing an SSH `session` after `nohup` background start often returns `io.EOF`; trap and ignore that condition while still checking real errors.

    ## Ansible YAML formatting fights
    - Keep `---` at top of YAML docs and ensure two spaces before comments. Prettier and `ansible-lint` can disagree; use a single formatter (Prettier) and align `.yamllint` config with it.

    ## Pre-commit: Node/Prettier hiccups
    - If Prettier fails to bootstrap Node, prefer using a local `node_modules/.bin/prettier` in hooks.

    ## Rubocop requires Ruby headers
    - Install `ruby-dev`/`ruby-devel` on your host if `rubocop` fails building native extensions.
