# >>> paddock shell integration >>>
# paddock shell integration
# Install: paddock shell-init fish | source

function claude
    paddock run $argv
end

function _paddock_chpwd --on-variable PWD
    set -l profile (paddock which --quiet 2>/dev/null)
    if test -n "$profile"
        set -gx CLAUDE_CONFIG_DIR "$HOME/.paddock/profiles/$profile"
    else
        set -e CLAUDE_CONFIG_DIR
    end
end

_paddock_chpwd
# <<< paddock shell integration <<<
