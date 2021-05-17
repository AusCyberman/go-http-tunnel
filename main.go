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
	h := httphandler{make(chan message,100), make(chan message,100), make(chan error), make(chan error)}
	return h
}

func (s httphandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	fmt.Println("got request")
	switch req.Method {
	case "PUT":
		buf := make([]byte, 100)
		if num, err := req.Body.Read(buf); err != nil {
			s.errors <- err
		} else if num == 0 {

		} else {
			s.http_input <- message{buf}

		}
	}
	fmt.Println("stuff")
	select {
	case available := <-s.tcp_input:
		fmt.Println("there is data")
		fmt.Println(len(available.message))
		writer.Write(available.message)
	default:
		writer.WriteHeader(http.StatusOK)
	}
}

func (s httphandler) process_http() {
	for {
		fmt.Printf("failed: %s", http.ListenAndServe("127.0.0.1:8080", s))
		fmt.Print("wtf")
	}
}

func (s httphandler) process_tcp() {
	socket, err := net.Listen("tcp", "127.0.0.1:65535")
	if err != nil {
		s.fatal_errors <- err
	}
	var client net.Conn
	for {
		if client == nil {
			client, err = socket.Accept()
			if err != nil {
				s.errors <- err
			}
			fmt.Println("got client")
		}
		select {
		case message := <-s.http_input:
			if n, err := client.Write(message.message); err != nil {
				s.errors <- err
				client = nil
			} else if n == 0 {
				s.errors <- errors.New("Sent no data")
			}
		default:
			buf := make([]byte, 10)
			if num, err := client.Read(buf); err != nil {
				fmt.Println("errors")
				s.errors <- err
				client = nil
			} else if num == 0 {
				fmt.Println("zero")
			} else {
				fmt.Println("found one")
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
		default:
		}
	}
}
