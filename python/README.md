breaks
    randomly choose 3 to 10 commands and jumble them up whilst maintaining everything
        so this would mean that ls would execute cp and cp would exectue something else, etc

chaostok/
├── agent/
│   ├── __init__.py
│   ├── engine.py          # Main logic: state → sabotage
│   ├── scheduler.py       # 3–7 minute random timer
│   ├── logger.py          # Hidden log writer
│   └── sandbox.py         # Sanity checks (e.g. is this a VM?)
├── sabotage_modules/
│   ├── __init__.py
│   ├── disable_aliases.py
│   ├── inject_bom.py
│   ├── break_cron.py
│   └── corrupt_apt.py
├── checker/
│   ├── verify.py          # Checker to validate repairs
│   └── report.py          # Score / output
├── config/
│   ├── rules.yaml         # Heuristics and sabotage weights
│   └── whitelist.yaml     # Never sabotage these paths/services
├── chaos.service          # Systemd unit file
├── start.sh               # Installer
├── README.md
└── requirements.txt

