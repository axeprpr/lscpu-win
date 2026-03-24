param(
    [Parameter(Mandatory = $true)]
    [string]$Owner,
    [Parameter(Mandatory = $true)]
    [string]$Repository,
    [Parameter(Mandatory = $true)]
    [string]$Tag
)

$ErrorActionPreference = "Stop"

$version = $Tag.TrimStart("v")
$assetName = "lscpu-win_${Tag}_windows_amd64.zip"
$headers = @{
    Accept = "application/vnd.github+json"
    "User-Agent" = "codex"
}
if ($env:WINGET_AUTH_TOKEN) {
    $headers.Authorization = "Bearer $($env:WINGET_AUTH_TOKEN)"
}

function Get-ReleaseAsset([string]$ApiUrl, [string]$WantedName) {
    for ($i = 0; $i -lt 30; $i++) {
        $release = Invoke-RestMethod -Uri $ApiUrl -Headers $headers
        $asset = $release.assets | Where-Object { $_.name -eq $WantedName } | Select-Object -First 1
        if ($asset) {
            return $asset
        }
        Start-Sleep -Seconds 10
    }
    throw "Timed out waiting for release asset $WantedName"
}

$releaseApiUrl = "https://api.github.com/repos/$Owner/$Repository/releases/tags/$Tag"
$zipAsset = Get-ReleaseAsset -ApiUrl $releaseApiUrl -WantedName $assetName
$shaAsset = Get-ReleaseAsset -ApiUrl $releaseApiUrl -WantedName "$assetName.sha256"
$assetUrl = $zipAsset.browser_download_url
$shaLine = (Invoke-WebRequest -Uri $shaAsset.browser_download_url -Headers $headers -UseBasicParsing).Content.Trim()
$sha256 = ($shaLine -split '\s+')[0].ToUpperInvariant()

$packageIdentifier = $env:PACKAGE_IDENTIFIER
$packageName = $env:PACKAGE_NAME
$publisher = $env:PUBLISHER
$shortDescription = $env:SHORT_DESCRIPTION
$moniker = $env:MONIKER
$license = $env:LICENSE
$licenseUrl = $env:LICENSE_URL
$homepage = $env:HOMEPAGE

New-Item -ItemType Directory -Force -Path out/winget | Out-Null

@"
PackageIdentifier: $packageIdentifier
PackageVersion: $version
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.9.0
"@ | Set-Content "out/winget/$packageIdentifier.yaml"

@"
PackageIdentifier: $packageIdentifier
PackageVersion: $version
PackageLocale: en-US
Publisher: $publisher
PackageName: $packageName
ShortDescription: $shortDescription
Moniker: $moniker
License: $license
LicenseUrl: $licenseUrl
Homepage: $homepage
ManifestType: defaultLocale
ManifestVersion: 1.9.0
"@ | Set-Content "out/winget/$packageIdentifier.locale.en-US.yaml"

@"
PackageIdentifier: $packageIdentifier
PackageVersion: $version
InstallerLocale: en-US
Platform:
- Windows.Desktop
InstallModes:
- silent
- silentWithProgress
Installers:
- Architecture: x64
  InstallerType: zip
  NestedInstallerType: portable
  NestedInstallerFiles:
  - RelativeFilePath: lscpu-win-amd64.exe
    PortableCommandAlias: lscpu-win
  InstallerUrl: $assetUrl
  InstallerSha256: $sha256
ManifestType: installer
ManifestVersion: 1.9.0
"@ | Set-Content "out/winget/$packageIdentifier.installer.yaml"
