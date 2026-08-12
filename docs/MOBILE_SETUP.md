# FormForge 1.4 Mobile PWA Setup

1. Install FormForge 1.8.0 on Windows and launch it from the Desktop/Start Menu shortcut.
2. Sign in as an administrator and open **Mobile**.
3. Enable private-LAN access, save, and restart FormForge.
4. Run `enable-lan-firewall.ps1` as administrator when Windows Firewall blocks the selected private-network port.
5. Download and trust the local FormForge CA certificate on the approved phone/tablet.
6. Connect the phone and PC to the same private Wi-Fi and open one of the HTTPS addresses displayed by FormForge.
7. Sign in with a separate approved account and use Add to Home Screen/Install App.

The PWA includes all responsive screens: Coaching Team, AI Coach, workouts, nutrition, Health + Plans, progress, habits, Community, check-ins, Security, Mobile, Administration, and Settings. When the Windows server cannot be reached, the installed PWA can open its cached phone-only offline coach. Shared records, source updates, backups, and collaboration require the PC server.

For remote use, use a private VPN or professionally managed HTTPS hosting. A local IP address does not work over cellular and the FormForge port should not be publicly forwarded.
