$ErrorActionPreference = 'Stop'
Get-NetFirewallRule -DisplayName "FormForge Private LAN HTTPS" -ErrorAction SilentlyContinue | Remove-NetFirewallRule
Write-Host "Removed the FormForge private-network firewall rule."
