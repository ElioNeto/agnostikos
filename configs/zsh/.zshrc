# =============================================================================
# .zshrc — Zsh configuration with Zinit plugin manager
# =============================================================================

# ---- Zinit (lightweight plugin manager) ----
ZINIT_HOME="${XDG_DATA_HOME:-${HOME}/.local/share}/zinit/zinit.git"
if [[ ! -f "${ZINIT_HOME}/zinit.zsh" ]]; then
  print -P "%F{14}▓ Installing Zinit…%f"
  command mkdir -p "${ZINIT_HOME}" && command chmod g-rwX "${ZINIT_HOME}"
  command git clone --depth=1 https://github.com/zdharma-continuum/zinit "${ZINIT_HOME}" \
    && print -P "%F{10}▓ Zinit installed%f" \
    || print -P "%F{9}▓ Failed to install Zinit%f"
fi
source "${ZINIT_HOME}/zinit.zsh"

# ---- Plugins loaded via Zinit ----
zinit light zsh-users/zsh-autosuggestions
zinit light zsh-users/zsh-syntax-highlighting
zinit light zsh-users/zsh-completions

# ---- fzf ----
# Load fzf (assumes fzf is installed via package manager)
if (( $+commands[fzf] )); then
  # Source fzf shell integration if available
  [[ -f /usr/share/fzf/key-bindings.zsh ]] && source /usr/share/fzf/key-bindings.zsh
  [[ -f /usr/share/fzf/completion.zsh ]]   && source /usr/share/fzf/completion.zsh

  # Fallback: check user-local fzf installation
  [[ -f "${XDG_CONFIG_HOME:-$HOME/.config}/fzf/fzf.zsh" ]] && \
    source "${XDG_CONFIG_HOME:-$HOME/.config}/fzf/fzf.zsh"
fi

# ---- Starship prompt ----
if (( $+commands[starship] )); then
  # Ensure pastel-powerline preset is used
  export STARSHIP_CONFIG="${STARSHIP_CONFIG:-${XDG_CONFIG_HOME:-$HOME/.config}/starship.toml}"
  if [[ ! -f "$STARSHIP_CONFIG" ]]; then
    command mkdir -p "$(dirname "$STARSHIP_CONFIG")"
    starship preset pastel-powerline -o "$STARSHIP_CONFIG" 2>/dev/null || true
  fi
  eval "$(starship init zsh)"
fi

# ---- Completion system ----
autoload -Uz compinit && compinit -d "${XDG_CACHE_HOME:-$HOME/.cache}/zsh/zcompdump"

# ---- History ----
HISTFILE="${HISTFILE:-${HOME}/.zhistory}"
HISTSIZE="${HISTSIZE:-10000}"
SAVEHIST="${SAVEHIST:-10000}"
setopt SHARE_HISTORY         # Share history across sessions
setopt HIST_IGNORE_DUPS      # Do not record duplicate entries
setopt HIST_IGNORE_SPACE     # Do not record commands starting with space
setopt HIST_REDUCE_BLANKS    # Remove superfluous blanks

# ---- Zsh options ----
setopt EXTENDED_GLOB         # Extended globbing (#, ~, ^)
setopt AUTO_CD               # cd by typing directory name
setopt AUTO_PUSHD            # Make cd push the old directory onto the stack
setopt PUSHD_IGNORE_DUPS     # No duplicate directories in stack
setopt NO_BEEP               # Disable beep on error
setopt LOCAL_OPTIONS         # Allow functions to have local options
setopt LOCAL_TRAPS           # Allow functions to have local traps

# ---- Key bindings ----
bindkey -e                    # Emacs keybindings (also enables Ctrl+R, Ctrl+T etc.)

# ---- Aliases ----
alias g='git'
alias lg='lazygit'
alias k='kubectl'
alias d='docker'
alias ag='agnostic'

# ---- Modern GNU replacements (guarded by command existence) ----
if (( $+commands[eza] )); then
  alias ls='eza --icons=always'
  alias ll='eza -la --icons=always'
  alias la='eza -a --icons=always'
  alias lt='eza --tree --icons=always'
fi
if (( $+commands[bat] )); then
  alias cat='bat'
fi
if (( $+commands[rg] )); then
  alias grep='rg'
fi
if (( $+commands[zoxide] )); then
  alias cd='z'
fi
if (( $+commands[dust] )); then
  alias df='dust'
  alias du='dust'
fi
if (( $+commands[procs] )); then
  alias ps='procs'
fi
if (( $+commands[fd] )); then
  alias find='fd'
fi
if (( $+commands[btop] )); then
  alias top='btop'
fi
if (( $+commands[httpie] )); then
  alias curl='httpie'
fi

# ---- Miscellaneous ----
