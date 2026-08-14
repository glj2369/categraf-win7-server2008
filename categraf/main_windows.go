//go:build windows

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/chai2010/winsvc"
	"github.com/kardianos/service"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"

	"flashcat.cloud/categraf/agent"
	"flashcat.cloud/categraf/config"
	"flashcat.cloud/categraf/pkg/pprof"
)

var (
	pprofStart          uint32
	flagWinSvcName      = flag.String("win-service-name", "categraf", "Set windows service name")
	flagWinSvcDesc      = flag.String("win-service-desc", "Categraf", "Set windows service description")
	flagWinSvcInstall   = flag.Bool("win-service-install", false, "Install windows service")
	flagWinSvcUninstall = flag.Bool("win-service-uninstall", false, "Uninstall windows service")
	flagWinSvcStart     = flag.Bool("win-service-start", false, "Start windows service")
	flagWinSvcStop      = flag.Bool("win-service-stop", false, "Stop windows service")
)

var (
	serviceAgent            *agent.Agent
	runningAsWindowsService bool
)

func shouldRunAsWindowsService() bool {
	isSvc, err := svc.IsWindowsService()
	if err == nil {
		return isSvc
	}
	interactive, err2 := svc.IsAnInteractiveSession()
	if err2 == nil {
		return !interactive
	}
	return false
}

func serveWindowsSCM() bool {
	if !shouldRunAsWindowsService() {
		return false
	}
	runningAsWindowsService = true
	// 必须立刻 svc.Run，不能先写日志或开 eventlog，否则 Win7 冷启动会 30 秒连不上 SCM。
	if err := svc.Run(*flagWinSvcName, &categrafWinService{}); err != nil {
		log.Fatalln("F! failed to run windows service:", err)
	}
	return true
}

type categrafWinService struct{}

func (p *categrafWinService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	go startAgentProcess()
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			stopWindowsService()
			return
		}
	}
	return
}

func startCategrafService(s service.Service) {
	const attempts = 4
	for i := 1; i <= attempts; i++ {
		err := s.Start()
		if err == nil {
			log.Println("I! start categraf service ok")
			return
		}
		if i < attempts && isSCMStartTimeout(err) {
			log.Printf("W! start categraf service failed (%d/%d): %v, retry in 3s\n", i, attempts, err)
			time.Sleep(3 * time.Second)
			continue
		}
		log.Println("E! start categraf service failed:", err)
		return
	}
}

func isSCMStartTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, windows.ERROR_SERVICE_REQUEST_TIMEOUT) {
		return true
	}
	var errno syscall.Errno
	if errors.As(err, &errno) && errno == windows.ERROR_SERVICE_REQUEST_TIMEOUT {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(err.Error(), "超时") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "timed out") ||
		strings.Contains(msg, "timely") ||
		strings.Contains(msg, "1053") ||
		strings.Contains(msg, "30000")
}

func stopWindowsService() {
	if serviceAgent != nil {
		serviceAgent.Stop()
	}
}

func runAgent(ag *agent.Agent) {
	serviceAgent = ag
	if runningAsWindowsService {
		if config.Config.Log.FileName != "stdout" && config.Config.Log.FileName != "stderr" &&
			config.Config.Log.FileName != "" {
			initLog(config.Config.Log.FileName)
		}
		ag.Start()
		handleSignal(ag)
		return
	}

	ag.Start()
	go profile()
	handleSignal(ag)
}

func doOSsvc() {
	// install service
	if *flagWinSvcInstall {
		if err := winsvc.InstallService(appPath, *flagWinSvcName, *flagWinSvcDesc); err != nil {
			log.Fatalln("F! failed to install service:", *flagWinSvcName, "error:", err)
		}
		fmt.Println("done")
		os.Exit(0)
	}

	// uninstall service
	if *flagWinSvcUninstall {
		if err := winsvc.RemoveService(*flagWinSvcName); err != nil {
			log.Fatalln("F! failed to uninstall service:", *flagWinSvcName, "error:", err)
		}
		fmt.Println("done")
		os.Exit(0)
	}

	// start service
	if *flagWinSvcStart {
		if err := winsvc.StartService(*flagWinSvcName); err != nil {
			log.Fatalln("F! failed to start service:", *flagWinSvcName, "error:", err)
		}
		fmt.Println("done")
		os.Exit(0)
	}

	// stop service
	if *flagWinSvcStop && runtime.GOOS == "windows" {
		if err := winsvc.StopService(*flagWinSvcName); err != nil {
			log.Fatalln("F! failed to stop service:", *flagWinSvcName, "error:", err)
		}
		fmt.Println("done")
		os.Exit(0)
	}
}

func profile() {
	// TODO: replace with windows event
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			file := filepath.Join(config.Config.ConfigDir, ".pprof")
			if _, err := os.Stat(file); err == nil {
				if !atomic.CompareAndSwapUint32(&pprofStart, 0, 1) {
					return
				}
				go pprof.Go()
			}
		}
	}
}
