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

type CmdString []string

func (c *CmdString) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*c = []string{single}
		return nil
	}

	var multi []string
	if err := json.Unmarshal(data, &multi); err == nil {
		*c = multi
		return nil
	}

	return fmt.Errorf("cmd is not string or []string")
}

type CommandInfo struct {
	Dir string    `json:"dir"`
	Cmd CmdString `json:"cmd"`
}

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

func readCommandsJSON(path string) (map[string]CommandInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var commands map[string]CommandInfo
	err = json.Unmarshal(data, &commands)
	return commands, err
}

func isWorkerOnline(addr string, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr+":"+port, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func sendCommand(name, addr, dir string, commands []string, port string, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}

	if !isWorkerOnline(addr, port, 2*time.Second) {
		fmt.Printf("[%s] Offline or unreachable (connection failed)\n", name)
		return
	}

	conn, err := net.Dial("tcp", addr+":"+port)
	if err != nil {
		fmt.Printf("[%s] Connection error: %v\n", name, err)
		return
	}
	defer conn.Close()

	req := struct {
		Dir string   `json:"dir"`
		Cmd []string `json:"cmd"`
	}{
		Dir: dir,
		Cmd: commands,
	}

	reqBytes, _ := json.Marshal(req)
	_, err = conn.Write(append(reqBytes, '\n'))
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
	metricsMode := flag.Bool("metrics", false, "Gets all worker metrics, prints and then exits.")

	filter := flag.String("filter", "", "Only target workers whose name includes this string")
	dirFlag := flag.String("dir", "", "Default directory to run command in on workers (if not overridden per-worker)")
	portFlag := flag.String("port", "8000", "Port to connect to workers on")
	flag.Parse()

	var wg sync.WaitGroup

	if *metricsMode {
		workerFile := flag.Arg(0)

		workers, err := readWorkers(workerFile)
		if err != nil {
			log.Fatalf("Failed to read workers file: %v", err)
		}
		for name, addr := range workers {
			wg.Add(1)
			go func(name, addr string) {
				defer wg.Done()
				sendCommand(name, addr, "", []string{"__get_metrics__"}, *portFlag, nil)
			}(name, addr)
		}
		wg.Wait()
		return
	}

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

		for name, info := range commands {
			if *filter != "" && !strings.Contains(name, *filter) {
				continue
			}
			addr, ok := workers[name]
			if !ok {
				fmt.Printf("[WARN] No address found for worker '%s', skipping\n", name)
				continue
			}

			dirToUse := info.Dir
			if dirToUse == "" {
				dirToUse = *dirFlag
			}

			wg.Add(1)
			go sendCommand(name, addr, dirToUse, info.Cmd, *portFlag, &wg)
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

		for name, addr := range workers {
			if *filter != "" && !strings.Contains(name, *filter) {
				continue
			}
			wg.Add(1)
			go sendCommand(name, addr, *dirFlag, []string{command}, *portFlag, &wg)
		}
		wg.Wait()
	}
}
