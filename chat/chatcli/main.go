package main

import (
	"io"
	"net"
	"os"

	log "github.com/sirupsen/logrus"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()
	go func() {
		_, _ = io.Copy(os.Stdout, conn)
	}()

	_, _ = io.Copy(conn, os.Stdout)
	log.Infof("%s: exit", conn.LocalAddr())
}
