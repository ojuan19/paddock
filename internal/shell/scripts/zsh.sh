# >>> paddock shell integration >>>
# paddock shell integration
# Install: eval "$(paddock shell-init zsh)" in ~/.zshrc

alias claude='paddock run'

_paddock_chpwd() {
  local profile
  profile=$(paddock which --quiet 2>/dev/null)
  if [[ -n "$profile" ]]; then
    export CLAUDE_CONFIG_DIR="$HOME/.paddock/profiles/$profile"
  else
    unset CLAUDE_CONFIG_DIR
  fi
}

(( ${chpwd_functions[(I)_paddock_chpwd]} )) || chpwd_functions+=(_paddock_chpwd)

_paddock_chpwd
# <<< paddock shell integration <<<
