package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

func readWorkers(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	workers := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ">", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		addr := strings.TrimSpace(parts[1])
		workers[name] = addr
	}
	return workers, scanner.Err()
}

func readCommandsJSON(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var commands map[string]string
	err = json.Unmarshal(data, &commands)
	return commands, err
}

func isWorkerOnline(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr+":8000", 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func sendCommand(name, addr, command string, wg *sync.WaitGroup) {
	defer wg.Done()

	if !isWorkerOnline(addr) {
		fmt.Printf("[%s] ❌ Offline or unreachable (connection failed)\n", name)
		return
	}

	conn, err := net.Dial("tcp", addr+":8000")
	if err != nil {
		fmt.Printf("[%s] Connection error: %v\n", name, err)
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte(command + "\n"))
	if err != nil {
		fmt.Printf("[%s] Failed to send command: %v\n", name, err)
		return
	}

	scanner := bufio.NewScanner(conn)
	fmt.Printf("== Output from %s ==\n", name)
	for scanner.Scan() {
		fmt.Printf("[%s] %s\n", name, scanner.Text())
	}
}

func main() {
	jsonMode := flag.Bool("json", false, "Read per-worker commands from commands.json")
	filter := flag.String("filter", "", "Only target workers whose name includes this string")
	flag.Parse()

	if *jsonMode {
		if len(flag.Args()) < 1 {
			fmt.Println("Usage: ./master -json workers.txt [--filter name]")
			return
		}
		workerFile := flag.Arg(0)
		commands, err := readCommandsJSON("commands.json")
		if err != nil {
			log.Fatalf("Failed to read commands.json: %v", err)
		}
		workers, err := readWorkers(workerFile)
		if err != nil {
			log.Fatalf("Failed to read workers file: %v", err)
		}

		var wg sync.WaitGroup
		for name, command := range commands {
			if *filter != "" && !strings.Contains(name, *filter) {
				continue
			}
			addr, ok := workers[name]
			if !ok {
				fmt.Printf("[WARN] No address found for worker '%s', skipping\n", name)
				continue
			}
			wg.Add(1)
			go sendCommand(name, addr, command, &wg)
		}
		wg.Wait()
	} else {
		if len(flag.Args()) < 2 {
			fmt.Println("Usage: ./master workers.txt \"<command>\" [--filter name]")
			return
		}
		workerFile := flag.Arg(0)
		command := flag.Arg(1)

		workers, err := readWorkers(workerFile)
		if err != nil {
			log.Fatalf("Failed to read workers file: %v", err)
		}

		var wg sync.WaitGroup
		for name, addr := range workers {
			if *filter != "" && !strings.Contains(name, *filter) {
				continue
			}
			wg.Add(1)
			go sendCommand(name, addr, command, &wg)
		}
		wg.Wait()
	}
}
