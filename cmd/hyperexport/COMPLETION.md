# Shell Completion Installation Guide

This guide explains how to install shell autocompletion for `hyperexport` and `hyperctl`.

## Quick Installation

### Bash

```bash
# hyperexport
hyperexport --completion bash | sudo tee /etc/bash_completion.d/hyperexport

# hyperctl
hyperctl completion -shell bash | sudo tee /etc/bash_completion.d/hyperctl

# Reload completions (or restart your shell)
source /etc/bash_completion.d/hyperexport
source /etc/bash_completion.d/hyperctl
```

### Zsh

```zsh
# Create completion directory if it doesn't exist
mkdir -p ~/.zsh/completion

# hyperexport
hyperexport --completion zsh > ~/.zsh/completion/_hyperexport

# hyperctl
hyperctl completion -shell zsh > ~/.zsh/completion/_hyperctl

# Add to ~/.zshrc if not already present
echo 'fpath=(~/.zsh/completion $fpath)' >> ~/.zshrc
echo 'autoload -Uz compinit && compinit' >> ~/.zshrc

# Reload
source ~/.zshrc
```

### Fish

```fish
# hyperexport
hyperexport --completion fish > ~/.config/fish/completions/hyperexport.fish

# hyperctl
hyperctl completion -shell fish > ~/.config/fish/completions/hyperctl.fish

# Completions are loaded automatically in fish
```

## Features

### hyperexport Completions

- **Command flags**: All 80+ command-line flags with descriptions
- **Value suggestions**:
  - `--format` → ovf, ova
  - `--provider` → vsphere, aws, azure, gcp, hyperv
  - `--encrypt-method` → aes256, gpg
  - `--manifest-target` → qcow2, raw, vdi
  - `--daemon-list` → all, running, completed, failed
- **Path completion**: File and directory completion for paths
- **Cloud paths**: Smart completion for S3, Azure, GCS, SFTP URLs

### hyperctl Completions

- **Commands**: submit, query, list, grep, rg, vm, status, cancel, migrate, etc.
- **Command-specific flags**: Each subcommand has its own flag suggestions
- **Dynamic completion**: Supports various input patterns

## Usage Examples

After installation, try typing:

```bash
# hyperexport - press TAB to see all options
hyperexport --<TAB>

# See format options
hyperexport --format <TAB>

# See provider options
hyperexport --provider <TAB>

# hyperctl - press TAB to see commands
hyperctl <TAB>

# See hyperctl submit options
hyperctl submit --<TAB>
```

## Verification

Test if completions are working:

```bash
# Should show available options
hyperexport --form<TAB>  # completes to --format
hyperctl comp<TAB>       # completes to completion

# Should show value suggestions
hyperexport --format <TAB>    # shows: ovf ova
hyperexport --provider <TAB>  # shows: vsphere aws azure gcp hyperv
```

## Troubleshooting

### Bash

If completions don't work immediately:

```bash
# Reload bash-completion
source /etc/bash_completion

# Or restart your shell
exec bash
```

### Zsh

If completions don't work:

```zsh
# Clear completion cache
rm -f ~/.zcompdump
compinit

# Or restart your shell
exec zsh
```

### Fish

Fish should load completions automatically. If not:

```fish
# Rebuild completion cache
fish_update_completions

# Or restart fish
exec fish
```

## System-wide vs User-specific

### System-wide (requires sudo)

**Bash**: `/etc/bash_completion.d/`
**Zsh**: `/usr/share/zsh/site-functions/`
**Fish**: `/usr/share/fish/vendor_completions.d/`

### User-specific (no sudo needed)

**Bash**: `~/.bash_completion`
**Zsh**: `~/.zsh/completion/`
**Fish**: `~/.config/fish/completions/`

## Manual Installation Example

```bash
# Generate and install manually
hyperexport --completion bash > hyperexport-completion.bash
sudo mv hyperexport-completion.bash /etc/bash_completion.d/hyperexport

# For user-specific installation
hyperexport --completion bash >> ~/.bash_completion
source ~/.bash_completion
```

## Additional Resources

- See `hyperexport --help` for all available flags
- See `hyperctl help` for all available commands
- Completion scripts are generated from the current binary, so they always match your installed version
