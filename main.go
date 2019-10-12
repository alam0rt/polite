package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os/exec"

	"net/http"
)

// Message is the type which we recieve from
// clients
type Message struct {
	Bool     bool
	Response Response
}

// Response is sent back to client
type Response struct {
	Exception string `json:"exception"`
}

// ExecFlag is a flag which when set will be exeuted when pods are running
var ExecFlag = flag.String("exec", "/bin/true", "help message for flagname")
var Host = flag.String("host", "localhost", "wait for this host before executing")
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

func matchHost(resolved []string) bool {
	for _, h := range resolved {
		if h == *Host {
			return true
		}
	}
	return false
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
	message.Response.Exception = "wow"
	raddr := r.RemoteAddr
	host, _, err := net.SplitHostPort(raddr)
	if err != nil {
		fmt.Printf("%s not a valid host?", raddr)
	}

	resolved := resolveHost(host)
	Incoming[host] = matchHost(resolved)

	if !Incoming[host] {
		fmt.Printf("Unwanted host: [%s]%s", host, raddr)
	}
	if checkHosts() {
		// everyone has phone in so now
		// we can exec
		message.Response.Exception = "done"
		fmt.Printf("All %v host(s) have phoned in...", len(Incoming))
		runCommand()
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
		json.NewEncoder(w).Encode(message.Response)
		handleMessage(r, &message)
	})
	http.ListenAndServe(":"+*ListenPort, nil)
}

func init() {
	// create a map with all the incoming clients
	Incoming = make(map[string]bool)
	flag.Parse()
}
