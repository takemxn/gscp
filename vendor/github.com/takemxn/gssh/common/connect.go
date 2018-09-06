package common
import (
	"golang.org/x/crypto/ssh"
	"fmt"
	"log"
	"net"
)
type ConnectInfo struct {
	Username   string
	Hostname   string
	Port       int
	Password string
}

func NewConnectInfo(user, host string, port int, password string) *ConnectInfo {
	return &ConnectInfo{user, host, port, password}
}
func (ci *ConnectInfo) Connect() (client *ssh.Client, err error){
	// Create client config
	config := &ssh.ClientConfig{
		User: ci.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(ci.Password),
		},
		HostKeyCallback: func(Hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	addr := fmt.Sprintf("%s:%d", ci.Hostname, ci.Port)
	client, err = ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Printf("ssh.Dial : %s", err)
		return
	}
	return
}
