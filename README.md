# essm

╭────────────╮
│  e ▸ ssm   │
╰────────────╯

Instead of guessing the instance ID or instance name to initiate a start-session why don't you just navigate and enter instances on a TUI.

NOTE: On build, not finished yet. This is just a PoC

For essm to work, you must be logged to aws cli. You can do so executing:

```bash
aws login
```

After that, running the TUI should work just fine.

Navigate through the table using arrow keys or vim-like basic navigation.

Current functionalities:

* "/" -> toggle search mode. Right now its just an input box, it doesn't find anything
* "ctrl+c" or "q" -> quit the TUI