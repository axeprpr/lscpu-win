# lscpu-win

`lscpu-win` is a small Windows-native replacement for `lscpu`, written in Go.

It reports:

- CPU architecture
- logical CPU count
- socket/core topology
- NUMA node count
- cache sizes
- vendor, model, family, stepping
- max CPU frequency from the Windows registry

## Build

```powershell
go build ./cmd/lscpu-win
```

## Run

```powershell
.\lscpu-win.exe
```

## GitHub Actions

This repository includes:

- `.github/workflows/release.yml` to build Windows binaries and create a GitHub release on tags like `v1.0.0`
- `.github/workflows/winget.yml` to generate a winget manifest from the release and open a PR against your `winget-pkgs` fork

## Repo Configuration

Set these repository variables before enabling the `winget.yml` workflow:

- `PACKAGE_IDENTIFIER` such as `YourName.lscpu-win`
- `PACKAGE_NAME` such as `lscpu-win`
- `PUBLISHER` such as `YourName`
- `SHORT_DESCRIPTION` such as `Windows-native lscpu replacement`
- `MONIKER` such as `lscpu-win`
- `LICENSE` such as `MIT`
- `LICENSE_URL` such as `https://github.com/<owner>/<repo>/blob/main/LICENSE`
- `HOMEPAGE` such as `https://github.com/<owner>/<repo>`
- `WINGET_FORK_OWNER` for your GitHub username that owns the `winget-pkgs` fork

Set these repository secrets:

- `WINGET_PKGS_PAT` with permission to push to your `winget-pkgs` fork and open pull requests

## Notes

Windows already exposes some CPU details through `systeminfo`, `wmic cpu`, PowerShell CIM, and Task Manager, but there is no built-in `lscpu` command.
