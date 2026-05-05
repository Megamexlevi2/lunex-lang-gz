while ($true) {
    Clear-Host
    Write-Host "NTL Installer"
    Write-Host ""
    Write-Host "1) Windows amd64"
    Write-Host "0) Exit"
    Write-Host ""

    $choice = Read-Host "Select an option"

    switch ($choice) {
        "1" {
            $asset = "ntl-windows-amd64.exe"
            $target = Join-Path $env:LOCALAPPDATA "Programs\ntl\ntl.exe"
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

    $tmp = [System.IO.Path]::GetTempFileName()
    Invoke-WebRequest -Uri "https://github.com/Megamexlevi2/ntl-go/releases/latest/download/$asset" -OutFile $tmp

    $dir = Split-Path $target -Parent
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    Move-Item -Force $tmp $target

    Write-Host ""
    Write-Host "Installed to: $target"
    Write-Host "Add this folder to PATH if needed:"
    Write-Host $dir
    break
}