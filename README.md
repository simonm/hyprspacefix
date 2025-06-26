# hyprspacefix

A command-line utility for bulk-assigning workspaces to monitors in the Hyprland window manager.

## Overview

`hyprspacefix` simplifies the process of assigning multiple workspaces to a specific monitor in Hyprland. Instead of manually configuring each workspace, you can assign an entire range of workspaces to a monitor with a single command.

I use this with [shikane](https://github.com/hw0lff/shikane) to ensure my workspaces are where I want them to be on a dual-monitor setup. This allow Xmonad type workspace management in Hyprland.

## Installation

### From Source

```bash
git clone https://github.com/simonm/hyprspacefix.git
cd hyprspacefix
go build
```

### Prerequisites

- Go 1.23.1 or later
- Hyprland window manager
- `hyprctl` must be available in your PATH

## Usage

```bash
hyprspacefix --name=<monitor-name> --range=<start>-<end> [options]
```

### Examples

Assign workspaces 1-5 to monitor DP-1:

```bash
hyprspacefix --name=DP-1 --range=1-5
```

Assign workspaces 10-20 to monitor HDMI-A-1 with verbose output:

```bash
hyprspacefix --name=HDMI-A-1 --range=10-20 --verbose
```

Preview commands without executing (dry run):

```bash
hyprspacefix --name=eDP-1 --range=1-3 --dry-run
```

### Options

- `--name`: The name of the monitor (required)
- `--range`: The range of workspaces to assign, format: `start-end` (required)
- `--dry-run`: Show what commands would be executed without running them
- `--verbose`: Enable detailed logging output
- `--help`: Display help information

## How It Works

`hyprspacefix` uses Hyprland's `hyprctl` command to:

1. Configure each workspace in the specified range to be associated with the target monitor
2. Move each workspace to that monitor
3. Execute all commands in a single batch for efficiency

The tool generates commands like:

```bash
hyprctl --batch "dispatch workspace 1 ; moveworkspacetomonitor 1 DP-1"
# and so on...
```

And then applies them in bulk.

## License

MIT Licence.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

