package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"formforge/internal/core"
)

//go:embed web/*
var embedded embed.FS

var version = "1.8.0"

func main() {
	if handleProtocolCommand() {
		return
	}
	dataDirFlag := flag.String("data-dir", "", "override application data directory")
	noOpen := flag.Bool("no-open", false, "do not open the desktop app window")
	serverOnly := flag.Bool("server-only", false, "run only the web server")
	portFlag := flag.Int("port", 0, "override server port")
	lanFlag := flag.Bool("lan", false, "temporarily bind to private-network interfaces")
	cloudFlag := flag.Bool("cloud", false, "run behind a managed HTTPS reverse proxy using plain HTTP")
	flag.Parse()

	cloud := *cloudFlag || envBool("FORMFORGE_CLOUD")
	dataDirOverride := strings.TrimSpace(*dataDirFlag)
	if dataDirOverride == "" {
		dataDirOverride = strings.TrimSpace(os.Getenv("FORMFORGE_DATA_DIR"))
	}
	dataDir, err := resolveDataDir(dataDirOverride)
	if err != nil {
		fatalDialog("FormForge startup failed", err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "logs"), 0700); err != nil {
		fatalDialog("FormForge startup failed", err.Error())
		return
	}
	logFile, err := os.OpenFile(filepath.Join(dataDir, "logs", "formforge.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}
	log.Printf("FormForge %s starting; data=%s", version, dataDir)

	store, err := core.OpenStore(dataDir)
	if err != nil {
		fatalDialog("FormForge database error", fmt.Sprintf("%v\n\nData was not deleted. See %s", err, filepath.Join(dataDir, "logs", "formforge.log")))
		return
	}
	masterKey, _, err := core.EnsureMasterKey(dataDir)
	if err != nil {
		fatalDialog("FormForge security error", err.Error())
		return
	}
	if err := applyEnvironmentConfiguration(store, masterKey); err != nil {
		fatalDialog("FormForge environment configuration error", err.Error())
		return
	}
	var certs core.CertPaths
	if !cloud {
		certs, err = core.EnsureCertificates(dataDir)
		if err != nil {
			fatalDialog("FormForge certificate error", err.Error())
			return
		}
	}
	webFS, _ := fs.Sub(embedded, "web")
	app := core.NewServer(store, masterKey, version, webFS, certs)
	app.Cloud = cloud
	app.TrustProxy = cloud
	app.SetupToken = strings.TrimSpace(os.Getenv("FORMFORGE_SETUP_TOKEN"))

	settings := core.Settings{Port: 8443}
	_ = store.Read(func(db core.Database) error { settings = db.Settings; return nil })
	port := settings.Port
	if cloud {
		if p, e := strconv.Atoi(strings.TrimSpace(os.Getenv("PORT"))); e == nil && p >= 1024 && p <= 65535 {
			port = p
		}
	}
	if *portFlag != 0 {
		port = *portFlag
	}
	if port == 0 {
		if cloud {
			port = 10000
		} else {
			port = 8443
		}
	}
	lan := settings.LANEnabled || *lanFlag || cloud
	bind := "127.0.0.1"
	if lan {
		bind = "0.0.0.0"
	}
	scheme := "https"
	if cloud {
		scheme = "http"
	}
	localURL := fmt.Sprintf("%s://127.0.0.1:%d", scheme, port)
	if !cloud && existingServer(localURL) {
		if !cloud && !*noOpen && !*serverOnly {
			_ = openAppWindow(localURL, certs.Cert)
		}
		return
	}
	selectedPort := port
	if !cloud {
		selectedPort, err = chooseAvailablePort(bind, port, 20)
	}
	if err != nil {
		fatalDialog("FormForge port error", fmt.Sprintf("%v\n\nClose any other FormForge processes or change the server port. Your data was not changed.\n\nLog: %s", err, filepath.Join(dataDir, "logs", "formforge.log")))
		return
	}
	if selectedPort != port {
		log.Printf("configured port %d was unavailable; using %d", port, selectedPort)
		port = selectedPort
		localURL = fmt.Sprintf("%s://127.0.0.1:%d", scheme, port)
		_ = store.Update(func(db *core.Database) error { db.Settings.Port = port; return nil })
	}
	httpServer := &http.Server{Addr: bind + ":" + strconv.Itoa(port), Handler: app.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20}

	errCh := make(chan error, 1)
	go func() {
		if cloud {
			log.Printf("HTTP listening behind trusted proxy on %s", httpServer.Addr)
			errCh <- httpServer.ListenAndServe()
			return
		}
		log.Printf("HTTPS listening on %s", httpServer.Addr)
		errCh <- httpServer.ListenAndServeTLS(certs.Cert, certs.Key)
	}()
	if err := waitForHealth(localURL, 10*time.Second); err != nil {
		select {
		case e := <-errCh:
			err = e
		default:
		}
		fatalDialog("FormForge could not start", fmt.Sprintf("The FormForge web service did not become ready.\n\n%v\n\nYour data remains in:\n%s\n\nSee the troubleshooting guide and log file.", err, dataDir))
		return
	}
	go automaticBackups(store, app)
	if !cloud && !*noOpen && !*serverOnly {
		if err := openAppWindow(localURL, certs.Cert); err != nil {
			log.Printf("app window failed: %v", err)
			fatalDialog("FormForge is running", fmt.Sprintf("The server started, but the app window could not open.\n\nOpen this address manually:\n%s\n\n%v", localURL, err))
		}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-stop:
		log.Printf("shutdown signal: %v", sig)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("server stopped: %v", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}

func envBool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

func applyEnvironmentConfiguration(store *core.Store, masterKey []byte) error {
	apiKey := strings.TrimSpace(os.Getenv("FORMFORGE_AI_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}
	var encryptedKey string
	var err error
	if apiKey != "" {
		if len(apiKey) > 500 {
			return fmt.Errorf("FORMFORGE_AI_API_KEY is too long")
		}
		encryptedKey, err = core.EncryptSecret(masterKey, apiKey)
		if err != nil {
			return fmt.Errorf("encrypt AI API key: %w", err)
		}
	}
	return store.Update(func(db *core.Database) error {
		if v := strings.TrimSpace(os.Getenv("FORMFORGE_AI_MODE")); v != "" {
			v = strings.ToLower(v)
			if v != "auto" && v != "online" && v != "offline" {
				return fmt.Errorf("FORMFORGE_AI_MODE must be auto, online, or offline")
			}
			db.Settings.AIMode = v
		}
		if v := strings.TrimSpace(os.Getenv("FORMFORGE_AI_BASE_URL")); v != "" {
			db.Settings.AIBaseURL = strings.TrimRight(v, "/")
		}
		if v := strings.TrimSpace(os.Getenv("FORMFORGE_AI_MODEL")); v != "" {
			db.Settings.AIModel = v
		}
		if encryptedKey != "" {
			db.Settings.AIAPIKeyEncrypted = encryptedKey
		}
		if v := strings.TrimSpace(os.Getenv("FORMFORGE_AGENT_BASE_URL")); v != "" {
			db.Settings.AgentBaseURL = strings.TrimRight(v, "/")
		}
		if v := strings.TrimSpace(os.Getenv("FORMFORGE_AGENT_MODEL")); v != "" {
			db.Settings.AgentModel = v
		}
		if v, ok := os.LookupEnv("FORMFORGE_AGENT_ENABLED"); ok {
			db.Settings.AgentEnabled = envBoolValue(v)
		}
		if v, ok := os.LookupEnv("FORMFORGE_AGENT_ALLOW_WEB"); ok {
			db.Settings.AgentAllowWeb = envBoolValue(v)
		}
		if v := strings.TrimSpace(os.Getenv("FORMFORGE_AGENT_SEARCH_URL")); v != "" {
			db.Settings.AgentSearchURL = v
		}
		if v := strings.TrimSpace(os.Getenv("FORMFORGE_AGENT_MAX_STEPS")); v != "" {
			n, e := strconv.Atoi(v)
			if e != nil || n < 1 || n > 20 {
				return fmt.Errorf("FORMFORGE_AGENT_MAX_STEPS must be between 1 and 20")
			}
			db.Settings.AgentMaxSteps = n
		}
		if v := strings.TrimSpace(os.Getenv("FORMFORGE_TAKEDOWN_CONTACT")); v != "" {
			db.Settings.TakedownContact = v
		}
		return nil
	})
}

func envBoolValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

func handleProtocolCommand() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(strings.ToLower(arg), "formforge://") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(arg), "formforge://restart") {
			exe, err := os.Executable()
			if err != nil {
				fatalDialog("FormForge restart failed", err.Error())
				return true
			}
			script := filepath.Join(os.TempDir(), fmt.Sprintf("FormForge-restart-%d.cmd", time.Now().UnixNano()))
			body := fmt.Sprintf("@echo off\r\ntimeout /t 1 /nobreak >nul\r\ntaskkill /IM FormForge.exe /T /F >nul 2>&1\r\ntimeout /t 1 /nobreak >nul\r\nstart \"\" \"%s\"\r\ndel \"%%~f0\"\r\n", strings.ReplaceAll(exe, "\"", "\"\""))
			if err := os.WriteFile(script, []byte(body), 0700); err != nil {
				fatalDialog("FormForge restart failed", err.Error())
				return true
			}
			_ = exec.Command("cmd.exe", "/C", "start", "", script).Start()
			return true
		}
		// formforge://start intentionally falls through to normal startup.
		return false
	}
	return false
}

func resolveDataDir(override string) (string, error) {
	if override != "" {
		return filepath.Abs(override)
	}
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", fmt.Errorf("LOCALAPPDATA is not set")
		}
		return filepath.Join(base, "FormForge"), nil
	}
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "FormForge"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "FormForge"), nil
}

func existingServer(url string) bool { return waitForHealth(url, 800*time.Millisecond) == nil }
func waitForHealth(url string, timeout time.Duration) error {
	client := &http.Client{Timeout: 700 * time.Millisecond, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/health")
		if err == nil {
			var health struct {
				Product string `json:"product"`
				OK      bool   `json:"ok"`
			}
			decodeErr := json.NewDecoder(resp.Body).Decode(&health)
			resp.Body.Close()
			if resp.StatusCode == 200 && decodeErr == nil && health.OK && health.Product == "formforge" {
				return nil
			}
			last = fmt.Errorf("port answered, but it was not FormForge")
		} else {
			last = err
		}
		time.Sleep(150 * time.Millisecond)
	}
	if last == nil {
		last = fmt.Errorf("health check timed out")
	}
	return last
}

func chooseAvailablePort(bind string, preferred, attempts int) (int, error) {
	if preferred < 1024 || preferred > 65535 {
		preferred = 8443
	}
	for i := 0; i <= attempts; i++ {
		p := preferred + i
		if p > 65535 {
			break
		}
		ln, err := net.Listen("tcp", net.JoinHostPort(bind, strconv.Itoa(p)))
		if err == nil {
			_ = ln.Close()
			return p, nil
		}
	}
	return 0, fmt.Errorf("no available HTTPS port was found between %d and %d", preferred, preferred+attempts)
}

func automaticBackups(store *core.Store, app *core.Server) {
	core.RunMaintenanceCycle(store, app.MasterKey)
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		core.RunMaintenanceCycle(store, app.MasterKey)
	}
}

func openAppWindow(url, certPath string) error {
	if runtime.GOOS == "windows" {
		edge := findEdge()
		if edge != "" {
			pin, _ := certificateSPKIPin(certPath)
			args := []string{"--app=" + url, "--start-maximized", "--no-first-run"}
			if pin != "" {
				args = append(args, "--ignore-certificate-errors-spki-list="+pin)
			}
			return exec.Command(edge, args...).Start()
		}
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", url).Start()
	}
	for _, name := range []string{"xdg-open", "gio"} {
		if p, err := exec.LookPath(name); err == nil {
			return exec.Command(p, url).Start()
		}
	}
	return fmt.Errorf("no browser launcher found")
}
func findEdge() string {
	candidates := []string{filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Microsoft", "Edge", "Application", "msedge.exe"), filepath.Join(os.Getenv("PROGRAMFILES"), "Microsoft", "Edge", "Application", "msedge.exe"), filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "Application", "msedge.exe")}
	for _, p := range candidates {
		if p != "" {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	if p, err := exec.LookPath("msedge.exe"); err == nil {
		return p
	}
	return ""
}
func certificateSPKIPin(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return "", fmt.Errorf("invalid certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}
	spki, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(spki)
	return base64.StdEncoding.EncodeToString(h[:]), nil
}
func fatalDialog(title, msg string) {
	log.Printf("%s: %s", title, msg)
	if runtime.GOOS == "windows" {
		safeTitle := strings.ReplaceAll(title, "'", "''")
		safeMsg := strings.ReplaceAll(msg, "'", "''")
		script := fmt.Sprintf("Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show('%s','%s')", safeMsg, safeTitle)
		_ = exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).Run()
		return
	}
	fmt.Fprintln(os.Stderr, title+": "+msg)
}
