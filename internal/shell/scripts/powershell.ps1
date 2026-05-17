# >>> paddock shell integration >>>
# EXPERIMENTAL. paddock shell integration
# Install: paddock shell-init powershell | Out-String | Invoke-Expression  (in $PROFILE)

function claude { paddock run @args }

$global:_PaddockOriginalPrompt = $function:prompt
function global:prompt {
    $profile = & paddock which --quiet 2>$null
    if ($profile) {
        $env:CLAUDE_CONFIG_DIR = "$HOME/.paddock/profiles/$profile"
    } else {
        Remove-Item Env:CLAUDE_CONFIG_DIR -ErrorAction SilentlyContinue
    }
    & $global:_PaddockOriginalPrompt
}

$profile = & paddock which --quiet 2>$null
if ($profile) {
    $env:CLAUDE_CONFIG_DIR = "$HOME/.paddock/profiles/$profile"
}
# <<< paddock shell integration <<<
