$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$Repository = "Shooa/ccauth"
$InstallDir = if ($env:CCAUTH_INSTALL_DIR) {
    $env:CCAUTH_INSTALL_DIR
} else {
    Join-Path $env:LOCALAPPDATA "Programs\ccauth"
}

$Release = Invoke-RestMethod `
    -Headers @{ Accept = "application/vnd.github+json"; "User-Agent" = "ccauth-installer" } `
    -Uri "https://api.github.com/repos/$Repository/releases/latest"
if (-not $Release -or -not $Release.tag_name) {
    throw "ccauth installer: GitHub API returned no release (rate limit or network?)"
}
$Tag = [string]$Release.tag_name
$Version = $Tag.TrimStart("v")
if (-not $Version -or $Tag -eq $Version) {
    throw "ccauth installer: could not determine the latest release"
}

switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { $TargetArch = "amd64" }
    "ARM64" { $TargetArch = "arm64" }
    default { throw "ccauth installer: unsupported architecture $env:PROCESSOR_ARCHITECTURE" }
}

$Archive = "ccauth_${Version}_windows_${TargetArch}.zip"
$DownloadBase = "https://github.com/$Repository/releases/download/$Tag"
$TemporaryDir = Join-Path ([System.IO.Path]::GetTempPath()) ("ccauth-install-" + [guid]::NewGuid().ToString("N"))

try {
    New-Item -ItemType Directory -Path $TemporaryDir | Out-Null
    $ArchivePath = Join-Path $TemporaryDir $Archive
    $SumsPath = Join-Path $TemporaryDir "SHA256SUMS"

    Write-Host "Downloading ccauth $Version for windows/$TargetArch..."
    Invoke-WebRequest -UseBasicParsing -Uri "$DownloadBase/$Archive" -OutFile $ArchivePath
    Invoke-WebRequest -UseBasicParsing -Uri "$DownloadBase/SHA256SUMS" -OutFile $SumsPath

    $ChecksumLine = Get-Content $SumsPath | Where-Object { $_ -match "\s(?:\./)?$([regex]::Escape($Archive))$" } | Select-Object -First 1
    if (-not $ChecksumLine) {
        throw "ccauth installer: release checksum is missing"
    }
    $Expected = ($ChecksumLine -split "\s+")[0].ToLowerInvariant()
    $Actual = (Get-FileHash -Algorithm SHA256 -Path $ArchivePath).Hash.ToLowerInvariant()
    if ($Actual -ne $Expected) {
        throw "ccauth installer: SHA-256 checksum mismatch"
    }

    $ExtractDir = Join-Path $TemporaryDir "extracted"
    Expand-Archive -Path $ArchivePath -DestinationPath $ExtractDir
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -Force -Path (Join-Path $ExtractDir "ccauth.exe") -Destination (Join-Path $InstallDir "ccauth.exe")

    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $PathEntries = @($UserPath -split ";" | Where-Object { $_ })
    if ($PathEntries -notcontains $InstallDir) {
        $NewPath = (@($PathEntries) + $InstallDir) -join ";"
        [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
        $env:Path = "$env:Path;$InstallDir"
        Write-Host "Added $InstallDir to the user PATH. Open a new terminal to use it."
    }
    Write-Host "Installed ccauth $Version to $InstallDir"
} finally {
    if (Test-Path $TemporaryDir) {
        Remove-Item -Recurse -Force $TemporaryDir
    }
}
