# gollama — Windows install via PowerShell
#   iwr -useb https://raw.githubusercontent.com/majidkorai/gollama/main/install.ps1 | iex

$repo = "majidkorai/gollama"

# Detect arch
$arch = switch ([Environment]::Is64BitOperatingSystem) {
    $true  { "amd64" }
    $false { Write-Host "32-bit not supported"; exit 1 }
}

Write-Host "gollama — installing for windows/$arch"

# Download pre-built binary
$url = "https://github.com/$repo/releases/latest/download/gollama-windows-$arch.exe"
$out = Join-Path $env:USERPROFILE "gollama.exe"

try {
    Invoke-WebRequest -Uri $url -OutFile $out -ErrorAction Stop
    Write-Host "Installed to $out"
    Write-Host "Add $env:USERPROFILE to your PATH or run: &`"$out`""
} catch {
    Write-Host "No pre-built binary available. Building from source (requires Go)..."
    
    $go = Get-Command go -ErrorAction SilentlyContinue
    if (-not $go) {
        Write-Host "Go is required. Install from https://go.dev/doc/install"
        exit 1
    }

    $tmp = Join-Path $env:TEMP "gollama-build"
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
    
    git clone --depth 1 "https://github.com/$repo.git" $tmp
    Push-Location $tmp
    go build -o gollama.exe .
    Pop-Location

    Move-Item (Join-Path $tmp "gollama.exe") $out -Force
    Remove-Item -Recurse -Force $tmp
    Write-Host "Installed to $out"
}
