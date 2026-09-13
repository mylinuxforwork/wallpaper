# ne-image-sorter

Sorts a folder of images by aspect ratio and resolution, moving everything a
configurable policy rejects into a second folder. Built for this wallpaper
collection, but the policy is general.

## Build

```
cd src
make build          # bin/ne-image-sorter
make install        # ~/.local/bin/ne-image-sorter
```

## Run

```
ne-image-sorter                                  # terminal interface
ne-image-sorter --sort --dry-run                 # report, move nothing
ne-image-sorter --sort                           # move, no interface
ne-image-sorter --source ~/pics --dest ~/pics/x  # override the saved folders
```

| Flag | Meaning |
|------|---------|
| `--sort` | Run headless and exit, for scripts and cron |
| `--dry-run` | With `--sort`, report what would move without moving it |
| `--source`, `--dest` | Override the saved directories for one run |
| `--config` | Configuration file, default `~/.config/ne-image-sorter/config.json` |
| `--log-dir` | Log directory, default `~/.cache/ne-image-sorter` |
| `--version` | Print the version and exit |

## How the policy works

Rules are checked top to bottom against each image. **The first rule that
matches decides**, and anything reaching the bottom takes the default. That
ordering is what lets one narrow move rule sit above broad keep rules.

The shipped policy is the one this collection was sorted with:

```
1. move if height <= 1080
2. keep if aspect == 16:9  (+/-1%)
3. keep if aspect == 16:10 (+/-1%)
   otherwise move
```

A rule tests `aspect`, `width`, or `height` with `==`, `!=`, `<`, `<=`, `>`, or
`>=`. Tolerance is a percentage and applies to `==` and `!=` only, which is what
makes `==` usable against real files: a 2912x1632 wallpaper is 0.37% off 16:9
and a strict comparison would reject it. Aspect values accept `16:9`, `16/10`,
or a decimal such as `1.7778`.

The default policy reproduces the collection's own 2025-12-17 hand sort on 34 of
its 35 moves and 115 of its 120 keeps. `TestDefaultPolicyReproducesTheManualSort`
pins that, so changing the shipped defaults breaks the build rather than
silently re-filing the library.

## Keys

`↑`/`↓` move, `enter` confirms, `esc` goes back, `q` or `ctrl+c` quits. Those
mean the same thing on every screen. Nothing moves until the preview is
confirmed with `y`, and an existing file is never overwritten: a name clash is
suffixed `-1`, `-2` and so on.

In **Rules**: `a` adds, `d` deletes, `J`/`K` reorder, `t` toggles the default,
`r` restores the shipped policy.

## Layout

```
cmd/ne-image-sorter/    entry point, flags, headless mode
internal/domain/        Image, Rule, Policy, Config. No I/O, no dependencies
internal/repository/    Images and Config interfaces, plus filesystem and JSON
internal/sorter/        Plan then Apply, over the repository interfaces
internal/tui/           Bubbletea screens
internal/logging/       zerolog to a file
```

The repository interfaces are what let the service and every screen be tested
against in-memory fakes. Only `internal/repository` touches the filesystem.

Logs go to a file and never to the console, which is both the house rule and a
hard requirement here: a terminal interface redraws continuously, so a stray
console write corrupts the display.
