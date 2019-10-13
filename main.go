package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"net/http"
)

// NAME is the name of the program
const NAME = "polite"

// SHELL is path to the shell
const SHELL = "/bin/sh"

// Set up logging
var (
	buf    bytes.Buffer
	logger = log.New(&buf, NAME+": ", log.Lshortfile)
)

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

// DebugFlag is a flag which enables debug logging
var DebugFlag = flag.Bool("debug", false, "set to enable debug logging")

// Incoming will save our incoming requests
var Incoming map[string]bool

// Hosts will hold the Host struct
var Hosts map[string]Host

func debugf(log ...string) {
	if *DebugFlag {
		logger.Printf("debug: %s\n", log)
		fmt.Print(&buf)
		buf.Reset()
	}
}

func resolveHost(ip string) []string {
	// resolveHost takes an IP and returns
	// a slice containing all resolved names
	resolved, err := net.LookupAddr(ip)
	debugf(resolved...)
	if err != nil {
		logger.Printf("%s cannot be resolved!\n", ip)
	}
	return resolved
}

func matchHost2(resolved []string) (string, bool) {
	// matchHost2 returns the matched hostname and a boolean
	// if hosts provided by the `-host` argument successfully
	// matches a resolved name
	for _, r := range resolved {
		debugf("Resolved: " + r)
		for h := range Hosts {
			debugf("Hosts: " + h)
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
	// checkHosts2 assumes all hosts in the `Hosts` construct
	// are unready. It then iterates over all of those hosts and
	// subtracts when one is ready. If all are ready, the counter
	// will be 0 and then we can return a boolean.
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
	// countReady is a global function
	// which simply returns the number
	// of ready hosts as an int
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
		logger.Printf("%s not a valid host", raddr)
		logger.Print(err)
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
		logger.Printf("%s not a known host\n", ip)
	}
	if checkHosts2() {
		// everyone has phone in so now we can exec
		logger.Printf("executing %s\n", *ExecFlag)
		runCommand()
	}
}

func politeExec(args ...string) *exec.Cmd {
	// politeExec if given a single argument will exec it direct
	// providing 2 or more arguments will construct a /bin/sh -c "args.." for execution
	var buffer bytes.Buffer
	name := args[0]
	if len(args) == 1 {
		return exec.Command(name)
	}
	for _, arg := range args {
		buffer.WriteString(arg + " ")
	}
	cmd := []string{
		"-c",
		buffer.String(),
	}
	return exec.Command(SHELL, cmd...)
}

func runCommand() {
	c := strings.Fields(*ExecFlag)
	cmd := politeExec(c...)
	err := cmd.Run()
	if err != nil {
		logger.Printf("ran %s with error: %v", *ExecFlag, err)
		fmt.Print(&buf)
		os.Exit(1)
	}
	fmt.Print(&buf)
	os.Exit(0)
}

func main() {
	if len(Hosts) == 0 {
		logger.Fatalf("no hosts provided!")
	}

	logger.Printf("started %s on :%s\n", NAME, *ListenPort)
	logger.Printf("politely waiting to execute %s", *ExecFlag)

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("%s requested status\n", r.RemoteAddr)
		fmt.Fprintf(w, "%v/%v\n", countReady(), len(Hosts))
		fmt.Print(&buf)
		buf.Reset()
	})
	// print logs and then clear buffer
	fmt.Print(&buf)
	buf.Reset()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var message Message
		json.NewDecoder(r.Body).Decode(&message)
		handleMessage(r, &message)
		for h := range Hosts {
			if Hosts[h].ready {
				logger.Printf("%s ready (%v/%v)\n", h, countReady(), len(Hosts))
			}
		}
		fmt.Print(&buf)
		buf.Reset()
		fmt.Fprintf(w, "%s\n", "pong")
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
