package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/template"
	"time"

	"golang.org/x/crypto/ssh"
)

/*
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
COMMANDS NECESSARY TO RUN BEORE RUNNING commandList
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PROVISION_WITH=plex_image.yml vagrant up --provision
	PROVISION_WITH=../../stressplex/install_wrk.yml vagrant provision

	VM=1 make adhoc
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Example usage:

	go run main.go vagrant localhost:2222 ~/.vagrant.d/insecure_private_key
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
*/

type wrkFields struct {
	Name           string
	AdhocName      string
	PlexHealthPort string
	WrkThreads     string
	MaxConnections string
	TestMinutes    string
	TargetRps      string
	Wrk2Address    string
}

func NewWrkFields() wrkFields {
	return wrkFields{
		Name:           "lets_gooooooo",
		AdhocName:      fmt.Sprintf("%d", time.Now().Nanosecond()),
		PlexHealthPort: "8080",
		WrkThreads:     "12",
		MaxConnections: "100",
		TestMinutes:    "1",
		TargetRps:      "90000",
		Wrk2Address:    "http://127.0.0.1:8080",
	}
}

var commandList = []string{
	"echo \"[restarting plex]\"",
	"sudo supervisorctl restart plexd",
	"while netstat -lnt | awk '$4 ~ /:{{.PlexHealthPort}}$/ {exit 1}'; do sleep 10; done", // waits for plexd to come up
	"echo \"[creating input directory]\"",
	"sudo mkdir /var/tmp/plex/stressplex/{{.AdhocName}} -p",
	"sudo cp ~/live.lua /var/tmp/plex/stressplex/{{.AdhocName}}/live.lua",
	"echo \"[running wrk2]\"",
	"sudo /usr/local/bin/wrk2 -s /var/tmp/plex/stressplex/{{.AdhocName}}/live.lua -t{{.WrkThreads}} -c{{.MaxConnections}} -d{{.TestMinutes}}m -R {{.TargetRps}} --latency {{.Wrk2Address}} > ~/results.txt",
	"sudo cat ~/results.txt",
	"echo \"[writing output]\"",
	"sudo mkdir /var/log/plex/stressplex/{{.AdhocName}} -p",
	"sudo cp ~/results.txt /var/log/plex/stressplex/{{.AdhocName}}/{{.Name}}_resp_results",
	"echo \"[cleaning up]\"",
	"sudo rm ~/results.txt",
	"sudo rm -rf /var/tmp/plex/stressplex/{{.AdhocName}}",
	"echo \"RESULTS LOCATED: /var/log/plex/stressplex/{{.AdhocName}}/{{.Name}}_resp_results\"",
}

func main() {
	auth := getAuth(os.Args)

	client, session, err := connectToHost(os.Args[1], os.Args[2], auth)

	if err != nil {
		log.Fatal(err)
	}

	defer client.Close()

	var commands string
	params := NewWrkFields()

	for _, cmd := range commandList {
		var b bytes.Buffer

		t := template.Must(template.New("command").Parse(cmd))
		t.Execute(&b, params)

		fmt.Println(b.String())

		commands += b.String() + ";"
	}
	out, err := session.CombinedOutput(commands[:len(commands)-1])

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(string(out))
}

func getAuth(args []string) ssh.AuthMethod {
	switch len(args) {
	case 3:
		var pass string

		fmt.Print("Password: ")
		fmt.Scanf("%s\n", &pass)

		return ssh.Password(pass)
	case 4:
		pemBytes, err := ioutil.ReadFile(args[3])

		if err != nil {
			log.Fatal(err)
		}

		signer, err := ssh.ParsePrivateKey(pemBytes)

		if err != nil {
			log.Fatalf("parse key failed:%v", err)
		}

		return ssh.PublicKeys(signer)
	default:
		log.Fatalf("Usage: %s <user> <host:port> <(optional) location of .pem file>", os.Args[0])
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
