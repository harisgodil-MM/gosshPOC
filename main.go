package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

func main() {
	auth := getAuth(os.Args)

	client, session, err := connectToHost(os.Args[1], os.Args[2], auth)

	if err != nil {
		log.Fatal(err)
	}

	defer client.Close()

	out, err := session.CombinedOutput(os.Args[3])

	if err != nil {
		log.Fatal(err)
	}

	log.Print(string(out))
}

func getAuth(args []string) ssh.AuthMethod {
	switch len(args) {
	case 4:
		var pass string

		fmt.Print("Password: ")
		fmt.Scanf("%s\n", &pass)

		return ssh.Password(pass)
	case 5:
		pemBytes, err := ioutil.ReadFile(args[4])

		if err != nil {
			log.Fatal(err)
		}

		signer, err := ssh.ParsePrivateKey(pemBytes)

		if err != nil {
			log.Fatalf("parse key failed:%v", err)
		}

		return ssh.PublicKeys(signer)
	default:
		log.Fatalf("Usage: %s <user> <host:port> <command> <(optional) location of .pem file>", os.Args[0])
		return nil
	}
}

func connectToHost(user, host string, auth ssh.AuthMethod) (*ssh.Client, *ssh.Session, error) {

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{auth},
	}

	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, session, nil
}
