package main

import (
	"testing"
	"os/exec"
)

func TestScp(t *testing.T){
	cmd := exec.Command("./scp_test.sh")
	err := cmd.Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}
}
