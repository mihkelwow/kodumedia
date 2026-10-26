//go:build windows

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mihkelwow/kodumedia/internal/adapters/system/shell"
	"github.com/mihkelwow/kodumedia/internal/adapters/system/singleinstance"
	"github.com/mihkelwow/kodumedia/internal/adapters/system/startup"
	"github.com/mihkelwow/kodumedia/internal/adapters/system/tray"
	"github.com/mihkelwow/kodumedia/internal/app"
	"github.com/mihkelwow/kodumedia/internal/buildinfo"
	"github.com/mihkelwow/kodumedia/internal/config"
)

type componentError struct {
	component string
	err       error
}

func main() {
	cfg := config.Load()
	closeLog := configureLogging(cfg.DataDir)
	defer closeLog()
	version := buildinfo.Version()
	executable, _ := os.Executable()
	slog.Info("KoduMedia starting",
		"version", version,
		"launch", launchMode(os.Args[1:]),
		"executable", executable,
		"data_directory", cfg.DataDir,
	)

	homeURL := localURL(cfg.Address, "/")
	settingsURL := localURL(cfg.Address, "/settings")
	instance, alreadyRunning, err := singleinstance.Acquire("Local\\KoduMedia.Server")
	if err != nil {
		slog.Error("acquire single-instance lock", "error", err)
		os.Exit(1)
	}
	if alreadyRunning {
		_ = shell.OpenURL(homeURL)
		return
	}
	defer instance.Close()

	application, err := app.New(context.Background(), cfg, version)
	if err != nil {
		slog.Error("initialize mediaserver", "error", err)
		os.Exit(1)
	}
	startupManager, err := startup.New()
	if err != nil {
		slog.Error("initialize startup integration", "error", err)
		os.Exit(1)
	}

	userExit := make(chan struct{}, 1)
	var systemTray *tray.Tray
	systemTray = tray.New(cfg.DataDir, tray.Callbacks{
		OpenHome: func() {
			if err := shell.OpenURL(homeURL); err != nil {
				slog.Error("open Kodu home", "error", err)
				systemTray.Notify("KoduMedia", "Brauseri avamine ebaõnnestus.", true)
			}
		},
		StartupEnabled: startupManager.Enabled,
		ToggleStartup: func() {
			enable := !startupManager.Enabled()
			if err := startupManager.SetEnabled(enable); err != nil {
				slog.Error("change login startup", "enabled", enable, "error", err)
				systemTray.Notify("KoduMedia", "Sisselogimisel käivitamise muutmine ebaõnnestus.", true)
				return
			}
			if enable {
				systemTray.Notify("KoduMedia", "Kodu käivitub nüüd Windowsi sisselogimisel.", false)
			} else {
				systemTray.Notify("KoduMedia", "Automaatne käivitamine eemaldati.", false)
			}
		},
		ScanAll: func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			started, skipped, err := application.ScanAllLibraries(ctx)
			if err != nil {
				slog.Error("queue all library scans", "started", started, "skipped", skipped, "error", err)
				systemTray.Notify("KoduMedia", "Kõiki meediakogusid ei saanud skannima panna.", true)
				return
			}
			if started == 0 {
				systemTray.Notify("KoduMedia", "Skannimiseks ei ole veel meediakaustu lisatud.", false)
				return
			}
			systemTray.Notify("KoduMedia", fmt.Sprintf("Skannimine käivitati %d meediakogus.", started), false)
		},
		OpenSettings: func() {
			if err := shell.OpenURL(settingsURL); err != nil {
				slog.Error("open Kodu settings", "error", err)
				systemTray.Notify("KoduMedia", "Seadete avamine ebaõnnestus.", true)
			}
		},
		Exit: func() {
			select {
			case userExit <- struct{}{}:
			default:
			}
		},
	})

	errors := make(chan componentError, 2)
	go func() { errors <- componentError{component: "server", err: application.Run()} }()
	go func() { errors <- componentError{component: "system tray", err: systemTray.Run()} }()
	if err := systemTray.WaitReady(); err != nil {
		slog.Error("initialize system tray", "error", err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	shutdown := false
	for !shutdown {
		select {
		case signal := <-signals:
			slog.Info("shutdown signal received", "signal", signal.String())
			shutdown = true
		case <-userExit:
			slog.Info("application exit requested from system tray")
			shutdown = true
		case <-application.StopRequested():
			slog.Info("application restart requested")
			shutdown = true
		case failure := <-errors:
			if failure.err != nil {
				slog.Error("component stopped unexpectedly", "component", failure.component, "error", failure.err)
			} else {
				slog.Info("component stopped", "component", failure.component)
			}
			// The web server is the application. A tray failure must not make
			// playback and LAN access disappear during Windows login.
			if failure.component != "system tray" {
				shutdown = true
			}
		}
	}

	systemTray.Close()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := application.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}

func configureLogging(dataDir string) func() {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
		return func() {}
	}
	path := filepath.Join(dataDir, "kodumedia.log")
	writer, err := newRotatingLogWriter(path, 10<<20)
	if err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
		return func() {}
	}
	logger := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	return func() { _ = writer.Close() }
}

type rotatingLogWriter struct {
	path    string
	maximum int64
	file    *os.File
	size    int64
	mu      sync.Mutex
}

func newRotatingLogWriter(path string, maximum int64) (*rotatingLogWriter, error) {
	writer := &rotatingLogWriter{path: path, maximum: maximum}
	if err := writer.open(); err != nil {
		return nil, err
	}
	if writer.size > maximum {
		if err := writer.rotate(); err != nil {
			_ = writer.file.Close()
			return nil, err
		}
	}
	return writer, nil
}

func (w *rotatingLogWriter) Write(payload []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, os.ErrClosed
	}
	if w.maximum > 0 && w.size+int64(len(payload)) > w.maximum {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	written, err := w.file.Write(payload)
	w.size += int64(written)
	return written, err
}

func (w *rotatingLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *rotatingLogWriter) open() error {
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	w.file = file
	w.size = info.Size()
	return nil
}

func (w *rotatingLogWriter) rotate() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}
	_ = os.Remove(w.path + ".1")
	if err := os.Rename(w.path, w.path+".1"); err != nil && !os.IsNotExist(err) {
		return err
	}
	return w.open()
}

func localURL(address, path string) string {
	address = strings.TrimSpace(address)
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		if strings.HasPrefix(address, ":") {
			port = strings.TrimPrefix(address, ":")
		} else {
			port = "8096"
		}
	}
	host = strings.Trim(host, "[]")
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "8096"
	}
	return (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, port), Path: path}).String()
}
