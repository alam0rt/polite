package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"

	"net/http"
)

// NAME is the name of the program
const NAME = "polite"

// Message is the type which we recieve from
// clients
type Message struct {
	Bool bool
}

// Host represents a remote client
type Host struct {
	remoteAddr string
	remotePort string
	request    Message
	ready      bool
}

// arrayFlags holds a slice of flags
type arrayFlags []string

// String satisfies the Value interface used by flags
func (i *arrayFlags) String() string {
	return "string"
}

// Set satisfies the Value interface used by flags
func (i *arrayFlags) Set(value string) error {
	*i = append(*i, strings.TrimSpace(value))
	return nil
}

// HostFlags holds the passed in host flags
// e.g. -host grafana -host mongodb.default.svc.cluster.local
var HostFlags arrayFlags

// ExecFlag is a flag which when set will be exeuted when pods are running
var ExecFlag = flag.String("exec", "/bin/true", "help message for flagname")

// ListenPort is a flag to set which port to listen on
var ListenPort = flag.String("port", "8080", "port to listen on")

// Incoming will save our incoming requests
var Incoming map[string]bool

// Hosts will hold the Host struct
var Hosts map[string]Host

func (h *Host) status() bool {
	return h.ready
}

func (h *Host) resolve() {
	if h.remoteAddr == "" {
		fmt.Print("bad")
	}
	resolved := resolveHost(h.remoteAddr)
	fmt.Print(resolved)

}

func resolveHost(ip string) []string {
	resolved, err := net.LookupAddr(ip)
	if err != nil {
		fmt.Printf("%s cannot be resolved!", ip)
	}
	return resolved
}

func matchHost2(resolved []string) (string, bool) {
	for _, r := range resolved {
		for h := range Hosts {
			if h == r {
				return h, true
			}
		}
	}
	return "", false
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

func checkHosts2() bool {
	// assume everoyne is unready
	unready := len(Hosts)
	for _, h := range Hosts {
		if h.ready {
			// if the host is ready we can subtract from our count
			unready--
		}
	}
	if unready == 0 {
		return true
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

func countReady() int {
	count := 0
	for host := range Hosts {
		if Hosts[host].ready {
			count++
		}
	}
	return count
}

func handleMessage(r *http.Request, message *Message) {
	// handleMessage takes an incoming request and determines if it satisfies our conditions
	raddr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(raddr)
	if err != nil {
		fmt.Printf("%s not a valid host?", raddr)
	}

	resolved := resolveHost(ip)
	name, isMatch := matchHost2(resolved)
	if isMatch {
		// if the host is a match let's fill out the struct with
		// the rest of the info
		Hosts[name] = Host{
			remoteAddr: ip,
			remotePort: port,
			ready:      true,
			request:    *message,
		}
	} else {
		fmt.Printf("%s not in list of hosts\n", resolved[0])
	}
	if checkHosts2() {
		// everyone has phone in so now we can exec
		fmt.Printf("All %v host(s) have phoned in...\n", len(Incoming))
		runCommand()
	} else {
		i := 0
		for _, v := range Hosts {
			if v.ready == true {
				i++
			}
		}
		fmt.Printf("%v/%v\n", i, len(Hosts))
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
	var (
		buf    bytes.Buffer
		logger = log.New(&buf, "logger: ", log.Lshortfile)
	)

	logger.Printf("started %s on :%s\n", NAME, *ListenPort)
	logger.Printf("politely waiting to execute %s", *ExecFlag)

	fmt.Print(&buf)
	http.HandleFunc("/decode", func(w http.ResponseWriter, r *http.Request) {
		var message Message
		json.NewDecoder(r.Body).Decode(&message)
		handleMessage(r, &message)
		fmt.Fprintf(w, "%s\n", "pong")
		for h := range Hosts {
			if Hosts[h].ready {
				logger.Printf("%s ready (%v/%v)\n", h, countReady(), len(Hosts))
			}
		}
		fmt.Print(&buf)
	})
	http.ListenAndServe(":"+*ListenPort, nil)
}

func init() {
	Incoming = make(map[string]bool)
	Hosts = make(map[string]Host)
	flag.Var(&HostFlags, "host", "hosts to wait for")
	flag.Parse()

	for _, host := range HostFlags {
		Incoming[host] = false
		Hosts[host] = Host{}
	}
}
