package main

import (
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"net"
	"fmt"
	"log"
	"os"
	"strings"
)

type Loc struct {
	Username string
	Hostname string
	Path string
	Port     int
	Password string
}
func (loc *Loc) IsRemote() bool {
	return len(loc.Hostname) != 0
}
func Connect(loc *Loc)(c *Client, err error){
	c = &Client{}
	if !loc.IsRemote(){
		info, _ := os.Stat(loc.Path)
		c.Loc = loc
		c.info = info
		return c, err
	}

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
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Printf("unable to connect: %s", err)
		return
	}

	client, err := sftp.NewClient(conn, sftp.MaxPacket(*SIZE))
	if err != nil {
		conn.Close()
		log.Fatalf("unable to start sftp subsytem: %v", err)
	}
	if loc.IsRemote() {
		session, err := conn.NewSession()
		if err != nil {
			log.Fatalf("New session error %v\n", err)
		}
		result, err := session.Output("echo " + loc.Path)
		if err != nil {
			log.Fatalf("New output error %v\n", err)
		}
		loc.Path = strings.TrimSpace(string(result))
		c.info, err = client.Lstat(loc.Path)
		if err != nil {
			log.Fatalf("Lstat error %v\n", err)
		}
	}else{
		info, err := os.Stat(loc.Path)
		if err != nil {
			log.Fatalf("stat error%v", info)
		}
	}
	c.ssh = conn
	c.sftp = client
	c.Loc = loc
	return 
}
