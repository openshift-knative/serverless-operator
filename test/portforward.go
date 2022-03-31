package test

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
)

type PortForwardType struct {
	LocalPort uint32
	cmd       *exec.Cmd
}

func PortForward(pod corev1.Pod, remotePort uint32) (*PortForwardType, error) {
	cmd := exec.Command("oc", "port-forward", "-n", pod.Namespace, pod.Name, fmt.Sprintf(":%d", remotePort))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	var localPort uint32
	portRegex := regexp.MustCompile(`Forwarding from [^:]+:([0-9]+) -> ([0-9]+)`)
	portChannel := make(chan uint32)
	scanner := bufio.NewScanner(stdout)
	go func() {
		readPort := false
		for scanner.Scan() {
			line := scanner.Text()
			if !readPort {
				submatch := portRegex.FindStringSubmatch(line)
				if submatch != nil {
					i, _ := strconv.Atoi(submatch[1])
					portChannel <- uint32(i)
					readPort = true
				}
			}
		}
	}()

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	select {
	case localPort = <-portChannel:
		break
	case <-time.After(30 * time.Second):
		cmd.Process.Signal(syscall.SIGTERM)
		go cmd.Wait()
		return nil, fmt.Errorf("timeout waiting for 'oc port-forward' output log local port number")
	}

	return &PortForwardType{
		LocalPort: localPort,
		cmd:       cmd,
	}, nil
}

func (portForward *PortForwardType) Close() error {
	err := portForward.cmd.Process.Signal(syscall.SIGTERM)
	go portForward.cmd.Wait()

	return err
}
