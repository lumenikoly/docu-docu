$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$Repository = "lumenikoly/toudocu"
$Version = if ($env:TOUDOCU_VERSION) { $env:TOUDOCU_VERSION } else { "latest" }
$DefaultInstallDir = Join-Path ([Environment]::GetFolderPath("LocalApplicationData")) "Programs\toudocu"
$InstallDir = if ($env:TOUDOCU_INSTALL_DIR) { $env:TOUDOCU_INSTALL_DIR } else { $DefaultInstallDir }
$NoModifyPath = if ($env:TOUDOCU_NO_MODIFY_PATH) { $env:TOUDOCU_NO_MODIFY_PATH } else { "0" }
$TempDir = $null
$StageFile = $null

function Fail([string]$Message) {
    throw "toudocu installer: $Message"
}

try {
    if ($env:OS -ne "Windows_NT") {
        Fail "unsupported operating system; use install.sh on Linux or macOS"
    }
    if ($Version -ne "latest" -and $Version -notmatch '^\d+\.\d+\.\d+(-rc\.[1-9]\d*)?$') {
        Fail "TOUDOCU_VERSION must be latest, X.Y.Z, or X.Y.Z-rc.N"
    }
    if ($NoModifyPath -ne "0" -and $NoModifyPath -ne "1") {
        Fail "TOUDOCU_NO_MODIFY_PATH must be 0 or 1"
    }

    $Architecture = try {
        [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    } catch {
        $null
    }
    if (-not $Architecture) {
        $ProcessorArchitecture = if ($env:PROCESSOR_ARCHITEW6432) {
            $env:PROCESSOR_ARCHITEW6432
        } else {
            $env:PROCESSOR_ARCHITECTURE
        }
        $Architecture = switch ($ProcessorArchitecture) {
            "AMD64" { "X64"; break }
            "ARM64" { "Arm64"; break }
            default { $ProcessorArchitecture }
        }
    }
    if ($Architecture -eq "X64" -and $env:PROCESSOR_ARCHITEW6432 -eq "ARM64") {
        $Architecture = "Arm64"
    }
    $Asset = switch ($Architecture) {
        "X64" { "toudocu-windows-amd64.exe"; break }
        "Arm64" { "toudocu-windows-arm64.exe"; break }
        default { Fail "unsupported Windows architecture: $Architecture; only AMD64 and ARM64 are published" }
    }

    $ReleaseUrl = if ($Version -eq "latest") {
        "https://github.com/$Repository/releases/latest/download"
    } else {
        "https://github.com/$Repository/releases/download/$Version"
    }

    $TempDir = Join-Path ([IO.Path]::GetTempPath()) ("toudocu-install-" + [Guid]::NewGuid().ToString("N"))
    [void](New-Item -ItemType Directory -Path $TempDir)
    $Downloaded = Join-Path $TempDir $Asset
    $Checksums = Join-Path $TempDir "checksums.txt"

    Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseUrl/checksums.txt" -OutFile $Checksums
    Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseUrl/$Asset" -OutFile $Downloaded

    $Pattern = '^(?<hash>[0-9A-Fa-f]{64})\s+\*?' + [Regex]::Escape($Asset) + '$'
    $ChecksumMatches = @(Get-Content -LiteralPath $Checksums | ForEach-Object {
        if ($_ -match $Pattern) { $Matches['hash'] }
    })
    if ($ChecksumMatches.Count -ne 1) {
        Fail "checksums.txt has no unique SHA-256 entry for $Asset"
    }
    $Expected = $ChecksumMatches[0].ToLowerInvariant()
    $Actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $Downloaded).Hash.ToLowerInvariant()
    if ($Actual -ne $Expected) {
        Fail "SHA-256 mismatch for $Asset"
    }

    $DownloadedVersion = (& $Downloaded version | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $DownloadedVersion -notmatch '^\d+\.\d+\.\d+$') {
        Fail "the downloaded binary cannot report a valid version"
    }
    $ExpectedVersion = $Version -replace '-rc\.\d+$', ''
    if ($Version -ne "latest" -and $DownloadedVersion -ne $ExpectedVersion) {
        Fail "the downloaded binary reported $DownloadedVersion, expected $ExpectedVersion"
    }

    [void](New-Item -ItemType Directory -Force -Path $InstallDir)
    $Target = Join-Path $InstallDir "toudocu.exe"
    $AlreadyInstalled = $false
    if (Test-Path -LiteralPath $Target -PathType Leaf) {
        $Installed = (Get-FileHash -Algorithm SHA256 -LiteralPath $Target).Hash.ToLowerInvariant()
        if ($Installed -eq $Expected) {
            $AlreadyInstalled = $true
        }
    }

    if (-not $AlreadyInstalled) {
        $StageFile = Join-Path $InstallDir (".toudocu.new." + [Guid]::NewGuid().ToString("N"))
        Copy-Item -LiteralPath $Downloaded -Destination $StageFile
        if (Test-Path -LiteralPath $Target -PathType Leaf) {
            $Backup = Join-Path $InstallDir (".toudocu.backup." + [Guid]::NewGuid().ToString("N"))
            [IO.File]::Replace($StageFile, $Target, $Backup, $true)
            $StageFile = $null
            Remove-Item -LiteralPath $Backup -Force -ErrorAction SilentlyContinue
        } else {
            [IO.File]::Move($StageFile, $Target)
            $StageFile = $null
        }
    }

    $PathChanged = $false
    $PathUpdateFailed = $false
    if ($InstallDir -eq $DefaultInstallDir) {
        $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
        $Entries = @($UserPath -split ';' | Where-Object { $_ })
        $AlreadyPresent = $Entries | Where-Object { $_.TrimEnd('\') -ieq $DefaultInstallDir.TrimEnd('\') }
        if (-not $AlreadyPresent -and $NoModifyPath -eq "0") {
            $NewUserPath = (($Entries + $DefaultInstallDir) -join ';')
            try {
                [Environment]::SetEnvironmentVariable("Path", $NewUserPath, "User")
                $PathChanged = $true
            } catch {
                $PathUpdateFailed = $true
                Write-Warning "Cannot update user PATH; add $DefaultInstallDir manually."
            }
        }
    }

    if ($AlreadyInstalled) {
        Write-Output "toudocu $DownloadedVersion is already installed at $Target"
    } else {
        Write-Output "Installed toudocu $DownloadedVersion at $Target"
    }
    if ($InstallDir -ne $DefaultInstallDir) {
        $CurrentEntries = @($env:Path -split ';')
        if (-not ($CurrentEntries | Where-Object { $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\') })) {
            Write-Output "Add $InstallDir to PATH to run toudocu by name."
        }
    } elseif ($NoModifyPath -eq "1" -or $PathUpdateFailed) {
        Write-Output "Add $DefaultInstallDir to PATH to run toudocu by name."
    } elseif ($PathChanged) {
        Write-Output "Open a new terminal to use toudocu by name."
    }
} catch {
    Write-Error $_.Exception.Message
    exit 1
} finally {
    if ($StageFile -and (Test-Path -LiteralPath $StageFile)) {
        Remove-Item -LiteralPath $StageFile -Force
    }
    if ($TempDir -and (Test-Path -LiteralPath $TempDir)) {
        Remove-Item -LiteralPath $TempDir -Recurse -Force
    }
}
