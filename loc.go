package main

import (
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"net"
	"fmt"
	"log"
	"os"
)

type Loc struct {
	Username string
	Hostname string
	Filename string
	Port     int
	Password string
}
func (loc *Loc) IsRemote() bool {
	return len(loc.Hostname) != 0
}
func Connect(loc *Loc)(c *Client, err error){
	// Create client config
	config := &ssh.ClientConfig{
		User: loc.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(loc.Password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	addr := fmt.Sprintf("%s:%d", loc.Hostname, loc.Port)
	ssh, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Printf("unable to connect: %s", err)
		return
	}

	sftp, err := sftp.NewClient(ssh, sftp.MaxPacket(*SIZE))
	if err != nil {
		ssh.Close()
		log.Fatalf("unable to start sftp subsytem: %v", err)
	}
	if loc.IsRemote() {
		c.FileInfo, err = sftp.Stat(loc.Filename)
		if err != nil {
			log.Fatalf("stat error")
		}
	}else{
		c.FileInfo, err = os.Stat(loc.Filename)
		if err != nil {
			log.Fatalf("stat error")
		}
	}
	c.ssh = ssh
	c.sftp = sftp
	c.Loc = loc
	return 
}
