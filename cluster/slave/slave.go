package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type CommandRequest struct {
	Dir string          `json:"dir"`
	Cmd json.RawMessage `json:"cmd"`
}

func collectMetrics() map[string]interface{} {
	cpuPercent, _ := cpu.Percent(0, false)
	vmStat, _ := mem.VirtualMemory()
	return map[string]interface{}{
		"cpu_percent": cpuPercent,
		"mem_total":   vmStat.Total,
		"mem_used":    vmStat.Used,
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	usr, err := user.Current()
	var homeDir string
	if err != nil || usr.HomeDir == "" {
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			log.Println("Cannot determine home directory, defaulting to current directory '.'")
			homeDir = "."
		}
	} else {
		homeDir = usr.HomeDir
	}

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			return
		} else if err != nil {
			log.Println("Read error:", err)
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var req CommandRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			log.Println("Failed to parse JSON command:", err)
			continue
		}

		dir := req.Dir
		if dir == "" {
			dir = homeDir
		}

		// Handle string or []string in "cmd"
		var commands []string
		if err := json.Unmarshal(req.Cmd, &commands); err != nil {
			var singleCmd string
			if err := json.Unmarshal(req.Cmd, &singleCmd); err != nil {
				log.Println("Failed to parse 'cmd' field:", err)
				continue
			}
			commands = []string{singleCmd}
		}

		for _, cmdStr := range commands {
			if cmdStr == "__get_metrics__" {
				metrics := collectMetrics()
				metricsJSON, _ := json.Marshal(metrics)
				writer.Write(metricsJSON)
				writer.WriteString("\n")
				writer.Flush()
				continue
			}

			log.Printf("Executing command in dir '%s': %s\n", dir, cmdStr)

			cmd := exec.Command("bash", "-c", cmdStr)
			cmd.Dir = dir

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				writer.WriteString("Error creating stdout pipe: " + err.Error() + "\n")
				writer.Flush()
				continue
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				writer.WriteString("Error creating stderr pipe: " + err.Error() + "\n")
				writer.Flush()
				continue
			}

			if err := cmd.Start(); err != nil {
				writer.WriteString("Error starting command: " + err.Error() + "\n")
				writer.Flush()
				continue
			}

			scanAndWrite := func(r io.Reader) {
				scanner := bufio.NewScanner(r)
				for scanner.Scan() {
					writer.WriteString(scanner.Text() + "\n")
				}
			}

			scanAndWrite(stdout)
			scanAndWrite(stderr)

			cmd.Wait()
			writer.Flush()
		}
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
