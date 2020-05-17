package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/platinummonkey/dconfig/daemon/store"
)

var options = struct {
	inMemory bool
	raftBindAddress string
	httpBindAddress string

	raftJoinAddress string
	nodeID string
} {
	raftBindAddress: ":8786",
	httpBindAddress: ":8081",
}

func init() {
	flag.BoolVar(&options.inMemory, "inMemory", false, "Use in-memory storage only")
	flag.StringVar(&options.raftBindAddress, "gossipAddr", options.raftBindAddress, "Set the gossip address")
	flag.StringVar(&options.httpBindAddress, "httpAddr", options.httpBindAddress, "Set the http server address, set to \"disabled\" to disable.")
	flag.StringVar(&options.raftJoinAddress, "join", "", "Set the join address if any")
	flag.StringVar(&options.nodeID, "id", "", "Node ID")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <data storage path> \n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "No storage directory specified\n")
		os.Exit(1)
	}

	// Ensure Raft storage exists.
	raftDir := flag.Arg(0)
	if raftDir == "" {
		fmt.Fprintf(os.Stderr, "No storage directory specified\n")
		os.Exit(1)
	}
	os.MkdirAll(raftDir, 0700)

	s := store.New(options.inMemory)
	s.RaftDir = raftDir
	s.RaftBind = options.raftBindAddress
	if err := s.Open(options.raftJoinAddress == "", options.nodeID); err != nil {
		log.Fatalf("failed to open store: %s", err.Error())
	}

	h := httpd.New(options.httpBindAddress, s)
	if err := h.Start(); err != nil {
		log.Fatalf("failed to start HTTP service: %s", err.Error())
	}

	// If join was specified, make the join request.
	if options.raftJoinAddress != "" {
		if err := join(options.raftJoinAddress, options.raftBindAddress, options.nodeID); err != nil {
			log.Fatalf("failed to join node at %s: %s", options.raftJoinAddress, err.Error())
		}
	}

	log.Println("dconfig started successfully")

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	log.Println("dconfig exiting")
}

func join(joinAddr, raftAddr, nodeID string) error {
	b, err := json.Marshal(map[string]string{"addr": raftAddr, "id": nodeID})
	if err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("http://%s/join", joinAddr), "application-type/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
