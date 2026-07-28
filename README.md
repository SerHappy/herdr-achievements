<p align="center">
  <img src="docs/logo.png" width="160" alt="Herdr Achievements mascot">
</p>

<h1 align="center">Herdr Achievements</h1>

<p align="center">
  <strong>Turn everyday agent milestones into tiny terminal celebrations.</strong>
</p>

<p align="center">
  A playful Herdr plugin that unlocks achievements as your AI agent herd gets to work.
</p>

## Your herd deserves trophies

Herdr already tells you what your agents are doing. Herdr Achievements makes those
moments feel a little more alive.

Install it, work as usual, and achievements unlock automatically when your herd
reaches a new milestone.

No prompts are read. No code is inspected. Nothing is sent over the network.

## Achievements

| | Achievement | How to unlock |
| --- | --- | --- |
| 🐾 | **FIRST HOOF** | Let the first agent join your herd. |
| 📦 | **FIRST DELIVERY** | Finish an agent turn. |
| 🚧 | **UNSTUCK** | Help an agent return from `blocked` to `working`. |
| 🐑🐑 | **DOUBLE TROUBLE** | Have two agents working at the same time. |
| 🐑🐑🐑 | **FULL HERD** | Have three agents working at the same time. |

Open the collection at any time to see your progress.

When an achievement unlocks, Herdr shows a native notification:

> 🏆 **Achievement unlocked**<br>
> FULL HERD — Three agents working at once

## Install

Requires Herdr 0.7.5+ and Go 1.22+.

```sh
herdr plugin install SerHappy/herdr-achievements --yes
```

Open your achievements:

```sh
herdr plugin action invoke open --plugin herdr-achievements
```

Or give the collection its own keybinding:

```toml
[[keys.command]]
key = "prefix+a"
type = "plugin_action"
command = "herdr-achievements.open"
description = "open achievements"
```

## Private by design

Herdr Achievements only listens to Herdr lifecycle events.

It stores locally:

* unlocked achievement IDs;
* unlock timestamps;
* current pane statuses;
* peak concurrent agent count.

It does not read or store prompts, terminal output, source code, filenames,
repository names, or agent conversations.

## Try the demo

Run the popup with a temporary completed collection:

```sh
./scripts/demo.sh
```

The demo does not modify your real achievement progress and requires no
dependencies beyond Go.

## Coming next

The next version will turn the collection into a proper trophy room:

* illustrated achievement cards;
* animated unlock reveals;
* keyboard navigation and detail views;
* responsive terminal layouts;
* shareable achievement cards.

## Development

```sh
go test ./...
go vet ./...
go build -o bin/herdr-achievements ./cmd/herdr-achievements
herdr plugin link .
```

## License

MIT
