package common
import (
	"golang.org/x/crypto/ssh"
	"fmt"
	"log"
	"net"
)
func Connect() (client *ssh.Client, err error){
	// Create client config
	config := &ssh.ClientConfig{
		User: Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(Password),
		},
		HostKeyCallback: func(Hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	addr := fmt.Sprintf("%s:%d", Hostname, Port)
	client, err = ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Printf("ssh.Dial : %s", err)
		return
	}
	return
}
func Connect2(user, host string, port int) (client *ssh.Client, err error){
	// Create client config
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(Password),
		},
		HostKeyCallback: func(host string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client, err = ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Printf("ssh.Dial : %s", err)
		return
	}
	return
}
