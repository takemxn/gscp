package scp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	sshcon "gssh/shared"
	"log"
)
var(
	srcFile string
	srcUser string
	srcHost string
)
func (scp *SecureCopier) copyLocalToRemote() (err error){
		errPipe := scp.errPipe
		outPipe := scp.outPipe
		srcFileInfo, err := os.Stat(srcFile)
		if err != nil {
			fmt.Fprintln(errPipe, "Could not stat source file "+srcFile)
			return err
		}
		procWriter := scp.dstWriter
		if scp.IsRecursive {
			if srcFileInfo.IsDir() {
				err = scp.processDir(procWriter, srcFile, srcFileInfo, outPipe, errPipe)
				if err != nil {
					fmt.Fprintln(errPipe, err.Error())
				}
			} else {
				err = scp.sendFile(procWriter, srcFile, srcFileInfo, outPipe, errPipe)
				if err != nil {
					fmt.Fprintln(errPipe, err.Error())
				}
			}
		} else {
			if srcFileInfo.IsDir() {
				return
			} else {
				err = scp.sendFile(procWriter, srcFile, srcFileInfo, outPipe, errPipe)
				if err != nil {
					fmt.Fprintln(errPipe, err.Error())
				}
			}
		}
		return
}
func (scp *SecureCopier) Copy(file, user, host string) (err error) {
	if host != "" {
		srcFile = file
		srcUser = user
		srcHost = host
	}else{
		err = scp.copyLocalToRemote()
		if err != nil {
			return
		}
	}
	return
}
