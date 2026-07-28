# Herdr Achievements

Give your AI agent herd a small reason to celebrate.

Herdr Achievements adds a calm terminal popup that tracks the little milestones of a productive session: the first agent, a finished delivery, getting unstuck, and a full herd working together.

## What it celebrates

| Achievement | Unlock condition |
| --- | --- |
| FIRST HOOF | The first agent is detected. |
| FIRST DELIVERY | An agent goes from `working` to `done` or `idle`. |
| UNSTUCK | An agent goes from `blocked` to `working`. |
| DOUBLE TROUBLE | Two agents are working at once. |
| FULL HERD | Three agents are working at once. |

## Install

Requires Herdr 0.7.5+ and Go 1.25+.

```sh
herdr plugin install SerHappy/herdr-achievements --yes
herdr plugin action invoke open --plugin herdr-achievements
```

Or bind it to a key:

```toml
[[keys.command]]
key = "prefix+a"
type = "plugin_action"
command = "herdr-achievements.open"
description = "open achievements"
```

## Preview

```text
HERDR ACHIEVEMENTS
5 / 5 unlocked

✓ FIRST HOOF
✓ FIRST DELIVERY
✓ UNSTUCK
✓ DOUBLE TROUBLE  2 / 2
✓ FULL HERD  3 / 3

Press Enter to close
```

Reproduce the preview locally with no dependencies beyond Go:

```sh
./scripts/demo.sh
```
