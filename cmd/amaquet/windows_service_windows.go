//go:build windows

package main

import (
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
)

type amaquetWindowsService struct {
	run func(<-chan struct{})
}

func runAsWindowsService(run func(<-chan struct{})) bool {
	isService, err := svc.IsWindowsService()
	if err != nil || !isService {
		return false
	}

	if err := svc.Run("Amaquet", amaquetWindowsService{run: run}); err != nil {
		os.Exit(1)
	}

	return true
}

func (s amaquetWindowsService) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	stop := make(chan struct{})
	done := make(chan struct{})

	status <- svc.Status{State: svc.StartPending}
	go func() {
		s.run(stop)
		close(done)
	}()
	status <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}

	for {
		select {
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				status <- svc.Status{
					State:   svc.Running,
					Accepts: svc.AcceptStop | svc.AcceptShutdown,
				}
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				close(stop)
				select {
				case <-done:
				case <-time.After(30 * time.Second):
				}
				status <- svc.Status{State: svc.Stopped}
				return false, 0
			}
		case <-done:
			status <- svc.Status{State: svc.Stopped}
			return false, 0
		}
	}
}
