$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$DistDir = Join-Path $RootDir "dist"
$DesktopDir = Join-Path $RootDir "desktop"
$WailsVersion = if ($env:WAILS_VERSION) { $env:WAILS_VERSION } else { "v2.12.0" }
$GoArch = (& go env GOARCH).Trim()
$GoPath = (& go env GOPATH).Trim()
$WailsBin = if ($env:WAILS_BIN) { $env:WAILS_BIN } else { Join-Path $GoPath "bin/wails.exe" }

function Resolve-ReleaseVersion {
    if ($env:RELEASE_VERSION) {
        $rawVersion = $env:RELEASE_VERSION
    } else {
        # long: Windows 包同样从主程序常量取版本，确保 CLI、桌面端和 Release 资产名三者一致。
        $line = Select-String -Path (Join-Path $RootDir "main.go") -Pattern '^const version = "m3u8dl-go ([^"]+)"$' | Select-Object -First 1
        if ($line) {
            $rawVersion = $line.Matches[0].Groups[1].Value
        } else {
            $rawVersion = "dev"
        }
    }

    if ([string]::IsNullOrWhiteSpace($rawVersion)) {
        return "dev"
    }
    if (-not $rawVersion.StartsWith("v") -and $rawVersion -ne "dev") {
        return "v$rawVersion"
    }
    return $rawVersion
}

$ReleaseVersion = Resolve-ReleaseVersion
$TargetName = "m3u8dl-go_${ReleaseVersion}_desktop_windows_${GoArch}"

function Test-DesktopArchive {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Archive,
        [Parameter(Mandatory = $true)]
        [string]$PackageName
    )

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [System.IO.Compression.ZipFile]::OpenRead($Archive)
    try {
        $entries = @($zip.Entries | ForEach-Object { $_.FullName.Replace("\", "/") })
    } finally {
        $zip.Dispose()
    }

    $required = @(
        "$PackageName/m3u8dl-go-desktop.exe",
        "$PackageName/m3u8dl-go-cli.exe"
    )
    foreach ($item in $required) {
        if ($entries -notcontains $item) {
            throw "桌面包缺少必要文件: $item"
        }
    }
    if ($entries | Where-Object { $_ -match '(^|/)[^/]+\.sha256$' }) {
        throw "桌面包不应包含 .sha256 文件"
    }
    Write-Output "桌面包结构检查通过: $Archive"
}

function Test-RunningOnWindows {
    $isWindowsVariable = Get-Variable -Name IsWindows -ErrorAction SilentlyContinue
    if ($isWindowsVariable) {
        return [bool]$isWindowsVariable.Value
    }
    return [System.Environment]::OSVersion.Platform -eq [System.PlatformID]::Win32NT
}

if (-not (Test-RunningOnWindows)) {
    throw "Windows desktop package must be built on Windows."
}

if (-not (Test-Path $WailsBin)) {
    & go install "github.com/wailsapp/wails/v2/cmd/wails@$WailsVersion"
}

$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $TmpDir | Out-Null
try {
    $CliHelper = Join-Path $TmpDir "m3u8dl-go-cli.exe"
    & go build -trimpath -ldflags="-s -w" -o $CliHelper $RootDir

    Remove-Item -Recurse -Force (Join-Path $DesktopDir "build") -ErrorAction SilentlyContinue
    Push-Location $DesktopDir
    try {
        & $WailsBin build -clean
    } finally {
        Pop-Location
    }

    $AppBin = Get-ChildItem -Path (Join-Path $DesktopDir "build/bin") -Filter "*.exe" -File | Select-Object -First 1
    if (-not $AppBin) {
        throw "未找到 Wails 生成的 Windows 桌面程序"
    }

    $PackageDir = Join-Path $TmpDir $TargetName
    New-Item -ItemType Directory -Path $PackageDir | Out-Null
    New-Item -ItemType Directory -Path $DistDir -Force | Out-Null
    Copy-Item $AppBin.FullName (Join-Path $PackageDir "m3u8dl-go-desktop.exe")
    Copy-Item $CliHelper (Join-Path $PackageDir "m3u8dl-go-cli.exe")
    Copy-Item (Join-Path $RootDir "README.md") (Join-Path $PackageDir "README.md")

    $Archive = Join-Path $DistDir "$TargetName.zip"
    Remove-Item -Force $Archive -ErrorAction SilentlyContinue
    Remove-Item -Force "$Archive.sha256" -ErrorAction SilentlyContinue
    Compress-Archive -Path $PackageDir -DestinationPath $Archive
    Test-DesktopArchive -Archive $Archive -PackageName $TargetName

    Write-Output $Archive
} finally {
    Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue
}
