package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("not enough args")
		os.Exit(1)
	}
	conn, err := net.Listen("tcp", fmt.Sprintf("%s:%s", os.Args[1], os.Args[2]))
	if err != nil {
		log.Fatalf("err: %e", err)
	}
loop:
	for {
		conn, err := conn.Accept()
		if err != nil {
			fmt.Printf("err: %e", err)
			goto loop
		}
		inp := make(chan bool)
		outp := make(chan bool)
		go func() {
			for {
				if _, err := io.Copy(conn, os.Stdin); err != nil {
					fmt.Printf("Error: %e", err)
					inp <- true
					break
				} else {
				}
			}
		}()
		go func() {
			for {
				if _, err := io.Copy(os.Stdout, conn); err != nil {
					fmt.Printf("Error: %e", err)
					outp <- true
					break
				} else {
				}
			}

		}()
		select {
		case <-inp:
			goto loop
		case <-outp:
			goto loop
		}

	}
}
