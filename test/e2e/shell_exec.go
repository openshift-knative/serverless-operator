package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

func execute(command string, t *testing.T) {
	t.Logf("Running command: %s", command)
	err, out, errout := shell(command)
	if out != "" {
		t.Log("--- stdout ---")
		t.Log(out)
	}
	if errout != "" {
		t.Log("--- stderr ---")
		t.Log(errout)
	}
	if err != nil {
		t.Fatalf("Error while running command: %v", err)
	}
}

func streamExecutionAndFailLate(command string, t *testing.T) {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		t.Errorf("Error while running command: %v", err)
	}
}

func shell(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}


