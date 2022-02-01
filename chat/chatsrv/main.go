package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"
)

type user struct {
	name string
	addr string
	msg  string
}

type client chan<- user

var (
	entering = make(chan client)
	leaving  = make(chan client)
	messages = make(chan user)
)

func main() {
	var err error
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	cfg := net.ListenConfig{
		KeepAlive: time.Minute,
	}

	l, err := cfg.Listen(ctx, "tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	go broadcaster(ctx)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := l.Accept()
				if err != nil {
					log.Warnf(err.Error())
					continue
				}
				go handleConn(conn)
			}
		}
	}(ctx)

	input := bufio.NewScanner(os.Stdin)
	u := user{name: "Admin"}
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				input.Scan()
				u.msg = u.name + ": " + input.Text()
				messages <- u
			}
		}
	}(ctx)

	<-ctx.Done()

	log.Infof("done")
	l.Close()
	log.Infof("exit")
}

func broadcaster(ctx context.Context) {
	clients := make(map[client]struct{})

	for {
		select {
		case <-ctx.Done():
			for cli := range clients {
				delete(clients, cli)
				close(cli)
			}

			close(entering)
			close(leaving)
			close(messages)

			return
		case msg := <-messages:
			for cli := range clients {
				cli <- msg
			}
		case cli := <-entering:
			clients[cli] = struct{}{}
		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
		}
	}
}

func handleConn(conn net.Conn) {
	ch := make(chan user)
	u := user{}
	go clientWriter(conn, ch)
	input := bufio.NewScanner(conn)

	u.addr = conn.RemoteAddr().String()
	fmt.Fprintln(conn, "Enter you name: ")
	input.Scan()
	u.name = input.Text()
	fmt.Fprintln(conn, u.name+" has arrived")

	u.msg = "You are " + u.name
	ch <- u
	messages <- u
	u.msg = u.name + " has arrived"
	entering <- ch

	log.Infof("%s has arrived", u.name)

	for input.Scan() {
		u.msg = u.name + ": " + input.Text()
		messages <- u
	}

	leaving <- ch
	u.msg = u.name + " has left"
	messages <- u
	log.Infof("%s has left", u.name)
	conn.Close()
}

func clientWriter(conn net.Conn, ch <-chan user) {
	for msg := range ch {
		fmt.Fprintf(conn, "%s %s\n", time.Now().Format("02 Jan 06 15:04 MST"), msg.msg)
	}
}
