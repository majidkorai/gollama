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

    # Install Visual C++ Redistributable (required by llama-server)
    if (-not (Test-Path "$env:SystemRoot\System32\VCRUNTIME140.dll")) {
        Write-Host "Installing Visual C++ Redistributable (required by llama-server)..."
        $vcUrl = "https://aka.ms/vs/17/release/vc_redist.x64.exe"
        $vcInstaller = Join-Path $env:TEMP "vc_redist.x64.exe"
        try {
            Invoke-WebRequest -Uri $vcUrl -OutFile $vcInstaller -ErrorAction Stop
            Start-Process -FilePath $vcInstaller -ArgumentList "/install /quiet /norestart" -Wait
            Write-Host "Visual C++ Redistributable installed."
        } catch {
            Write-Host "Warning: could not install Visual C++ Redistributable."
            Write-Host "If llama-server fails to run, install manually from:"
            Write-Host "  https://aka.ms/vcredist"
        }
        Remove-Item $vcInstaller -ErrorAction SilentlyContinue
    }

    # Add to user PATH so 'gollama' works in any terminal
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$env:USERPROFILE*") {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$env:USERPROFILE", "User")
        $env:Path += ";$env:USERPROFILE"
    }

    Write-Host "Installed to $out"
    Write-Host "gollama is ready. Open a new terminal and run: gollama"
    Write-Host "(or restart your current terminal)"
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
