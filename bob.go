package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
)

type message struct {
	//	ack_num int32
	// seq_num int32
	message []byte
}

type httphandler struct {
	http_input   chan message
	tcp_input    chan message
	errors       chan error
	fatal_errors chan error
}

func new_httphandler() httphandler {
	h := httphandler{make(chan message), make(chan message), make(chan error), make(chan error)}
	return h
}

func (s httphandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	fmt.Println("got request")
	switch req.Method {
	case "POST":
		var buf []byte
		if num, err := req.Body.Read(buf); err != nil {
			s.errors <- err
		} else if num == 0 {
			s.errors <- errors.New("Empty error")
		} else {

			s.http_input <- message{buf}
		}

	}
	select {
	case available := <-s.tcp_input:

		writer.Write(available.message)
	default:
		writer.WriteHeader(http.StatusOK)
	}
}

func (s httphandler) process_http() {
	for {
		fmt.Print("wtf")
		fmt.Printf("failed: %s", http.ListenAndServe("127.0.0.1:80", s))
	}
}

func (s httphandler) process_tcp() {
	socket, err := net.Listen("tcp", "127.0.0.1:22")
	if err != nil {
		s.fatal_errors <- err
		err = nil
	}
	var client net.Conn
	for {
		if client == nil {
			client, err = socket.Accept()
			if err != nil {
				s.errors <- err
				err = nil
			}
		}
		select {
		case message := <-s.http_input:
			if n, err := client.Write(message.message); err != nil {
				s.errors <- err
			} else if n == 0 {
				s.errors <- errors.New("Sent no data")
			}
		default:
			var buf []byte
			if num, err := client.Read(buf); err != nil {
				s.errors <- err
			} else if num == 0 {
			} else {
				s.tcp_input <- message{buf}
			}
		}
	}
}

func main() {
	handler := new_httphandler()
	go handler.process_http()
	go handler.process_tcp()
	defer close(handler.errors)
	defer close(handler.fatal_errors)
	defer close(handler.http_input)
	defer close(handler.tcp_input)

	for {
		select {
		case erro := <-handler.errors:
			log.Printf("Received error %e\n", erro)
		case fatal := <-handler.fatal_errors:
			log.Fatalf("Received fatal error %e", fatal)
		}
	}
}
