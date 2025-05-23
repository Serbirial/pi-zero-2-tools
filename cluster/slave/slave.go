package slave

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
			log.Println("Connection closed.")
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

		// Parse Cmd as either []string or string
		var commands []string
		if err := json.Unmarshal(req.Cmd, &commands); err != nil {
			// Not an array, try single string
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
				conn.Write(metricsJSON)
				conn.Write([]byte("\n"))
				continue
			}

			log.Printf("Executing command in dir '%s': %s\n", dir, cmdStr)

			cmd := exec.Command("bash", "-c", cmdStr)
			cmd.Dir = dir

			output, err := cmd.CombinedOutput()
			if err != nil {
				output = append(output, []byte("\nError: "+err.Error())...)
			}

			conn.Write(output)
			conn.Write([]byte("\n")) // newline after output
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
