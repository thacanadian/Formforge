param(
  [int]$Port = 8443,
  [string]$Program = "$env:LOCALAPPDATA\Programs\FormForge\FormForge.exe"
)
$ErrorActionPreference = 'Stop'
if (-not (Test-Path $Program)) { throw "FormForge.exe was not found at $Program" }
$name = "FormForge Private LAN HTTPS"
Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue | Remove-NetFirewallRule
New-NetFirewallRule -DisplayName $name -Direction Inbound -Action Allow -Protocol TCP -LocalPort $Port -Program $Program -Profile Private -RemoteAddress LocalSubnet | Out-Null
Write-Host "Created private-network firewall rule for TCP $Port, local subnet only."
