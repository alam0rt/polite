package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os/exec"
	"strings"

	"net/http"
)

// Message is the type which we recieve from
// clients
type Message struct {
	Bool bool
}

// arrayFlags holds a slice of flags
type arrayFlags []string

func (i *arrayFlags) String() string {
	return "string"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, strings.TrimSpace(value))
	return nil
}

var HostFlags arrayFlags

// ExecFlag is a flag which when set will be exeuted when pods are running
var ExecFlag = flag.String("exec", "/bin/true", "help message for flagname")
var ListenPort = flag.String("port", "8080", "port to listen on")

// Incoming will save our incoming requests
var Incoming map[string]bool

func resolveHost(ip string) []string {
	resolved, err := net.LookupAddr(ip)
	if err != nil {
		fmt.Printf("%s cannot be resolved!", ip)
	}
	return resolved
}

func matchHost(resolved []string) (string, bool) {
	for _, r := range resolved {
		for h := range Incoming {
			if h == r {
				return h, true
			}
		}
	}
	return "", false
}

func checkHosts() bool {
	for _, h := range Incoming {
		if h == false {
			return false
		}
	}
	return true
}

func handleMessage(r *http.Request, message *Message) {
	// handleMessage takes an incoming request and determines if it satisfies our conditions
	raddr := r.RemoteAddr
	host, _, err := net.SplitHostPort(raddr)
	if err != nil {
		fmt.Printf("%s not a valid host?", raddr)
	}

	resolved := resolveHost(host)
	matchedHost, isMatch := matchHost(resolved)
	if isMatch {
		Incoming[matchedHost] = true
	} else {
		fmt.Printf("%s not in list of hosts\n", resolved[0])
	}
	if checkHosts() {
		// everyone has phone in so now
		// we can exec
		fmt.Printf("All %v host(s) have phoned in...\n", len(Incoming))
		runCommand()
		// Should cleanly exit now and pass execution onto target program
	} else {
		i := 0
		for _, v := range Incoming {
			if v == true {
				i++
			}
		}
		fmt.Printf("%v/%v\n", i, len(Incoming))
	}

}

func runCommand() {
	cmd := exec.Command(*ExecFlag)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("ran %s with error: %v", *ExecFlag, err)
	}
}

func main() {
	http.HandleFunc("/decode", func(w http.ResponseWriter, r *http.Request) {
		var message Message
		json.NewDecoder(r.Body).Decode(&message)
		handleMessage(r, &message)
		fmt.Fprintf(w, "%s", "pong")
	})
	http.ListenAndServe(":"+*ListenPort, nil)
}

func init() {
	// create a map with all the incoming clients
	Incoming = make(map[string]bool)
	flag.Var(&HostFlags, "host", "hosts to wait for")
	flag.Parse()

	for _, host := range HostFlags {
		Incoming[host] = false
	}
}
