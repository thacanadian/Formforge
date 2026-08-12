package core

import (
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

func (s *Server) mobileInfo(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if s.Cloud {
		host := strings.TrimSpace(r.Host)
		if host == "" {
			host = "formforge.onrender.com"
		}
		jsonOut(w, 200, map[string]any{
			"enabled": true, "hosted": true, "urls": []string{"https://" + host},
			"currentUrl": "https://" + host, "caUrl": "", "requiresRestart": false,
			"instructions": []string{
				"Open this hosted FormForge address in Chrome or Safari.",
				"Sign in with the account created by the administrator.",
				"Use Add to Home Screen or Install App to install the PWA.",
				"The hosted server must remain active and its persistent disk must stay attached.",
			},
		})
		return
	}
	var settings Settings
	_ = s.Store.Read(func(db Database) error { settings = db.Settings; return nil })
	port := settings.Port
	if _, rawPort, err := net.SplitHostPort(r.Host); err == nil {
		if actual, err := strconv.Atoi(rawPort); err == nil && actual >= 1024 && actual <= 65535 {
			port = actual
		}
	}
	if port == 0 {
		port = 8443
	}
	urls := mobileURLs(port)
	currentHost := r.Host
	if currentHost == "" {
		currentHost = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	}
	jsonOut(w, 200, map[string]any{
		"enabled":         settings.LANEnabled,
		"port":            port,
		"urls":            urls,
		"currentUrl":      "https://" + currentHost,
		"caUrl":           "/api/system/ca",
		"requiresRestart": true,
		"instructions": []string{
			"Enable Mobile/LAN access and restart FormForge on the Windows PC.",
			"Keep the Windows FormForge process running while another device uses shared data.",
			"Install the FormForge local CA certificate on the phone or tablet.",
			"Open one of the private-network URLs while connected to the same Wi-Fi.",
			"Use the browser Add to Home Screen or Install App command.",
		},
	})
}

func mobileURLs(port int) []string {
	set := map[string]bool{}
	if host, _ := os.Hostname(); strings.TrimSpace(host) != "" {
		host = strings.TrimSpace(host)
		set["https://"+net.JoinHostPort(host, strconv.Itoa(port))] = true
		if !strings.Contains(host, ".") {
			set["https://"+net.JoinHostPort(host+".local", strconv.Itoa(port))] = true
		}
	}
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || (!ip.IsPrivate() && !ip.IsLinkLocalUnicast()) {
				continue
			}
			if v4 := ip.To4(); v4 != nil {
				set["https://"+net.JoinHostPort(v4.String(), strconv.Itoa(port))] = true
			}
		}
	}
	urls := make([]string, 0, len(set))
	for u := range set {
		urls = append(urls, u)
	}
	sort.Strings(urls)
	return urls
}
