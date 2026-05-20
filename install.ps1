$repo = "https://github.com/Megamexlevi2/ntl-lang-gz/releases/latest/download"
$asset = "ntl-windows-amd64.exe"
$target = Join-Path $env:LOCALAPPDATA "Programs\ntl\ntl.exe"

while ($true) {
    Clear-Host
    Write-Host "NTL Installer"
    Write-Host ""
    Write-Host "Windows"
    Write-Host ""
    Write-Host "1) Install"
    Write-Host "0) Exit"
    Write-Host ""

    $choice = Read-Host "Select an option"

    switch ($choice) {
        "1" {
            break
        }
        "0" {
            exit 0
        }
        default {
            Write-Host "Invalid option."
            Start-Sleep -Seconds 1
            continue
        }
    }

    $tmp = [System.IO.Path]::GetTempFileName() + ".exe"
    $url = "$repo/$asset"

    Write-Host "Downloading $url ..."
    try {
        Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing
    } catch {
        Write-Host "Download failed: $_"
        Remove-Item -Force $tmp -ErrorAction SilentlyContinue
        exit 1
    }

    $dir = Split-Path $target -Parent
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    Move-Item -Force $tmp $target

    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($userPath -notlike "*$dir*") {
        [Environment]::SetEnvironmentVariable("PATH", "$userPath;$dir", "User")
        Write-Host "Added $dir to your user PATH."
        Write-Host "Restart your terminal to apply the change."
    }

    Write-Host ""
    Write-Host "Installed: $target"
    Write-Host "Run: ntl help"
    break
}