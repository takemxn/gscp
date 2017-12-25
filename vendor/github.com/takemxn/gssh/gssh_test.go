package main

import (
	"testing"
)

func Test_parseArg(t *testing.T) {
	args := []string{"pssh"}
	t.Logf("%q:\n", args)
	err := parseArg(args)
	if err == nil {
		t.Fatal("error")
	}
	t.Log(err)

	args = []string{"pssh", "-p"}
	t.Logf("%q:\n", args)
	err = parseArg(args)
	if err == nil {
		t.Fatal("error")
	}
	t.Log(err)

	args = []string{"pssh", "-p", "password"}
	t.Logf("%q:\n", args)
	err = parseArg(args)
	if err == nil {
		t.Fatal("error")
	}
	t.Log(err)

	args = []string{"pssh", "-p", "password", "-t"}
	t.Logf("%q:\n", args)
	err = parseArg(args)
	if err == nil {
		t.Fatal("error")
	}
	t.Log(err)

	args = []string{"pssh", "-p", "password", "root"}
	t.Logf("%q:\n", args)
	err = parseArg(args)
	if err != nil {
		t.Fatal(err)
	}
	putInfo(t)

	args = []string{"pssh", "-p", "password", "root@"}
	t.Logf("%q:\n", args)
	err = parseArg(args)
	if err == nil {
		t.Fatal("error")
	}
	t.Log(err)

	args = []string{"pssh", "-p", "password", "root@localhost"}
	t.Logf("%q:\n", args)
	err = parseArg(args)
	if err != nil {
		t.Fatal("error")
	}
	putInfo(t)

	args = []string{"pssh", "-p", "password", "root@localhost:"}
	t.Logf("%q:\n", args)
	err = parseArg(args)
	if err == nil {
		t.Fatal(err)
	}
	t.Log(err)

	args = []string{"pssh", "-p", "password", "root@localhost:ab"}
	t.Logf("%q:\n", args)
	err = parseArg(args)
	if err == nil {
		t.Fatal(err)
	}
	t.Log(err)

	args = []string{"pssh", "-p", "password", "localhost:22"}
	t.Logf("%q:\n", args)
	err = parseArg(args)
	if err != nil {
		t.Fatal(err)
	}
	putInfo(t)

	args = []string{"pssh", "-p", "password", "root@localhost:22"}
	t.Logf("%q:\n", args)
	err = parseArg(args)
	if err != nil {
		t.Fatal("error")
	}
	putInfo(t)
}
func putInfo(t *testing.T) {
	t.Logf("user:%q", user)
	t.Logf("password:%q", password)
	t.Logf("hostname:%q", hostname)
	t.Logf("port:%d", port)
}
func Test_sshShell(t *testing.T) {
	err := sshShell("root", "root00x", "localhost", 22)
	if err != nil {
		t.Fatal(err)
	}
}
