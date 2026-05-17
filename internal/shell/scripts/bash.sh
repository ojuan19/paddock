# >>> paddock shell integration >>>
# paddock shell integration
# Install: eval "$(paddock shell-init bash)" in ~/.bashrc

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

: "${_paddock_last_pwd:=}"
_paddock_prompt() {
  if [[ "$PWD" != "$_paddock_last_pwd" ]]; then
    _paddock_last_pwd="$PWD"
    _paddock_chpwd
  fi
}

case "$PROMPT_COMMAND" in
  *_paddock_prompt*) ;;
  *) PROMPT_COMMAND="_paddock_prompt;${PROMPT_COMMAND}" ;;
esac

_paddock_chpwd
# <<< paddock shell integration <<<
