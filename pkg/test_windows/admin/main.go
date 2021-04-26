// +build windows

package main

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func main() {
	// if not elevated, relaunch by shellexecute with runas verb set
	var err error
	if len(os.Args) == 1 && os.Args[0] == "daemon-foreground" {
		if windows.GetCurrentProcessToken().IsElevated() {
			err = me()
		} else {
			err = errors.New("must run using admin privileges")
		}
	} else {
		err = runElevated()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error from me(): %v", err)
		os.Exit(1)
	}
}

func runElevated() error {
	if windows.GetCurrentProcessToken().IsElevated() {
		return me()
	}

	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	verbPtr, _ := windows.UTF16PtrFromString("runas")
	exePtr, _ := windows.UTF16PtrFromString(exe)
	cwdPtr, _ := windows.UTF16PtrFromString(cwd)
	argPtr, _ := windows.UTF16PtrFromString("daemon-foreground")
	err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, windows.SW_HIDE)
	if err != nil {
		return err
	}
	fmt.Println("Elevation succeeded, exiting")
	return nil
}

func IsAdmin() (bool, error) {
	stdout, err := os.Create("G:\\foo.txt")
	var sid *windows.SID

	// Directly copied from the official windows documentation. The Go API for this is a
	// direct wrap around the official C++ API.
	// See https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-checktokenmembership
	err = windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		fmt.Fprintf(stdout, "SID Error: %s", err)
		return false, err
	}
	return windows.GetCurrentProcessToken().IsMember(sid)
}

func me() error {
	stdout, err := os.Create("G:\\foo.txt")
	if err != nil {
		return err
	}
	adm, err := IsAdmin()
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Elevated?", windows.GetCurrentProcessToken().IsElevated())
	fmt.Fprintln(stdout, "Admin?", adm)
	return nil
}
