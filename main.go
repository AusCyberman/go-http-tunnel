package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

type ProcessType int

const (
	CLIENT ProcessType = iota
	SERVER
)

type message struct {
	//	ack_num int32
	//	seq_num int32
	seq_length int64
	message    []byte
}

var EMPTY_MESSAGE = message{0, []byte{}}

type httphandler struct {
	http_input   chan message
	tcp_input    chan message
	errors       chan error
	fatal_errors chan error
	ack_num      atomic.Value
	seq_num      atomic.Value
	h_type       ProcessType
}

func new_httphandler(t ProcessType) httphandler {
	h := httphandler{make(chan message, 100), make(chan message, 100), make(chan error), make(chan error), atomic.Value{}, atomic.Value{}, t}
	h.ack_num.Store(0)
	h.seq_num.Store(0)
	return h
}

func (s httphandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	fmt.Println("got request")
	switch req.Method {
	case http.MethodPost:
		buf := make([]byte, 10)
		if num, err := req.Body.Read(buf); err != nil && !errors.Is(err, io.EOF) {
			println(err)
			s.errors <- err
		} else if num != 0 {
			fmt.Println("data: ", string(buf[:num]))
			println("length is", num)
			s.http_input <- message{int64(num), buf}
		}
	}
	fmt.Println("stuff")
	select {
	case available := <-s.tcp_input:
		fmt.Println("there is data")
		if _, err := writer.Write(available.message); err != nil {
			s.errors <- err
		}
	default:
		writer.WriteHeader(http.StatusOK)
	}
}

func (s httphandler) fetch_http() error {
	for {

		data := EMPTY_MESSAGE
		select {
		case mes := <-s.tcp_input:
			data = mes

		case <-time.After(100 * time.Millisecond):
		}
		buf := bytes.NewBuffer(data.message[:data.seq_length])
		if resp, err := http.Post("http://127.0.0.1:8080", "text", buf); err != nil && !errors.Is(err, io.EOF) {
			s.errors <- err
		} else {
			buf := make([]byte, 100)
			if num, err := resp.Body.Read(buf); err != nil && !errors.Is(err, io.EOF) {
				s.errors <- err
			} else if num != 0 {
				fmt.Println(string(buf[:num]))
				s.http_input <- message{int64(num), buf}
			}

		}
	}
}

func (s httphandler) process_http() {
	switch s.h_type {
	case SERVER:
		for {
			s.errors <- http.ListenAndServe("127.0.0.1:8080", s)
		}
	case CLIENT:
		s.fetch_http()
	}
}

func (s httphandler) process_tcp() {
	var sock_string string
	switch s.h_type {
	case SERVER:
		sock_string = "127.0.0.1:65535"
	case CLIENT:
		sock_string = "127.0.0.1:65534"
	}
	var err error
	var client net.Conn
	for {
		if client == nil {
			client, err = net.Dial("tcp", sock_string)
			if err != nil {
				s.errors <- err
			}
			fmt.Println("got client")
		}
		select {
		case message := <-s.http_input:
			if n, err := client.Write(message.message[:message.seq_length]); err != nil {
				s.errors <- err
				client = nil
			} else if n == 0 {
				s.errors <- errors.New("Sent no data")
			}
		default:
			buf := make([]byte, 100)
			if num, err := client.Read(buf); err != nil && !errors.Is(err, io.EOF) {
				fmt.Println("errors")
				s.errors <- err
				client = nil
			} else if num != 0 {
				fmt.Println("found one")
				s.tcp_input <- message{int64(num), buf}
			}
		}
	}
}

func main() {
	var handler httphandler
	if len(os.Args) < 2 {
		fmt.Println("Using server")
		handler = new_httphandler(SERVER)
	} else {
		switch os.Args[1] {
		case "server":
			handler = new_httphandler(SERVER)
		case "client":
			handler = new_httphandler(CLIENT)
		}
	}
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
