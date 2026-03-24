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
$assetUrl = "https://github.com/$Owner/$Repository/releases/download/$Tag/$assetName"
$shaUrl = "$assetUrl.sha256"
$shaLine = (Invoke-WebRequest -Uri $shaUrl -UseBasicParsing).Content.Trim()
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
