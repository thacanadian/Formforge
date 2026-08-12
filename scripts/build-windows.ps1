param([switch]$SkipTests)
$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root
$Version = (Get-Content VERSION -Raw).Trim()
New-Item -ItemType Directory -Force dist, installer/bootstrap/payload | Out-Null
Remove-Item installer/bootstrap/payload/* -Force -ErrorAction SilentlyContinue
Set-Content installer/bootstrap/payload/BUILD_PAYLOAD.txt 'Build scripts replace this placeholder with the verified installer payload.'
$env:CGO_ENABLED='0'; $env:GOOS='windows'; $env:GOARCH='amd64'
go build -trimpath -ldflags "-s -w -H=windowsgui -X main.version=$Version" -o dist/FormForge.exe .
go build -trimpath -ldflags "-s -w -H=windowsgui" -o dist/Uninstall.exe ./installer/uninstaller
Remove-Item installer/bootstrap/payload/BUILD_PAYLOAD.txt -Force
Copy-Item dist/FormForge.exe installer/bootstrap/payload/FormForge.exe -Force
Copy-Item dist/Uninstall.exe installer/bootstrap/payload/Uninstall.exe -Force
Copy-Item README.md,VERSION,CHANGELOG.md,LICENSE.txt,installer/formforge.ico installer/bootstrap/payload -Force
Copy-Item docs/*.md installer/bootstrap/payload -Force
Copy-Item scripts/enable-lan-firewall.ps1,scripts/disable-lan-firewall.ps1 installer/bootstrap/payload -Force
if (-not $SkipTests) {
  go test ./...
  go vet ./...
  if (Get-Command node -ErrorAction SilentlyContinue) {
    node --check web/app.js
    node --check web/v14.js
    node --check web/v15.js
    node --check web/v16.js
    node --check web/v17.js
    node --check web/coaching-ui.js
    node --check web/sw.js
  }
}
go build -trimpath -ldflags "-s -w -H=windowsgui" -o dist/Setup.exe ./installer/bootstrap
Write-Host "Built dist\FormForge.exe, dist\Setup.exe, and dist\Uninstall.exe"
