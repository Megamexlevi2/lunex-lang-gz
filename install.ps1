$repo = "https://github.com/Megamexlevi2/lunex-lang-gz/releases/latest/download"
$asset = "lunex-windows-amd64.exe"
$target = Join-Path $env:LOCALAPPDATA "Programs\lunex\lunex.exe"

while ($true) {
    Clear-Host
    Write-Host "Lunex Installer"
    Write-Host ""
    Write-Host "Windows"
    Write-Host ""
    Write-Host "1) Install"
    Write-Host "0) Exit"
    Write-Host ""

    $choice = Read-Host "Select an option"
    $install = $false

    switch ($choice) {
        "1" {
            $install = $true
        }
        "0" {
            exit 0
        }
        default {
            Write-Host "Invalid option."
            Start-Sleep -Seconds 1
        }
    }

    if (-not $install) {
        continue
    }

    $tempName = [System.IO.Path]::GetRandomFileName() + ".exe"
    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) $tempName
    $url = "$repo/$asset"

    Write-Host "Downloading $url ..."
    try {
        Invoke-WebRequest -Uri $url -OutFile $tmp
    } catch {
        Write-Host "Download failed: $($_.Exception.Message)"
        Remove-Item -Force $tmp -ErrorAction SilentlyContinue
        exit 1
    }

    $dir = Split-Path $target -Parent
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    Move-Item -Force $tmp $target

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $paths = @()
    if ($userPath) {
        $paths = $userPath -split ';' | Where-Object { $_ -and $_.Trim() }
    }

    if ($paths -notcontains $dir) {
        $newPath = ($paths + $dir) -join ';'
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Host "Added $dir to your user PATH."
        Write-Host "Restart your terminal to apply the change."
    }

    Write-Host ""
    Write-Host "Installed: $target"
    Write-Host "Run: lunex help"
    break
}