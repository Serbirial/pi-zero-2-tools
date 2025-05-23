package watchdog

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// Read workers from file
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

// FIXME
func alertWorkerDown(name, addr string) {
}

func isOnline(addr string, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(addr, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func main() {
	// CLI flags
	var (
		port          string
		timeoutSec    int
		checkInterval int
	)
	flag.StringVar(&port, "port", "8000", "Port that all workers will be using.")
	flag.IntVar(&timeoutSec, "timeout", 2, "Timeout seconds for TCP dial.")
	flag.IntVar(&checkInterval, "interval", 10, "Interval seconds between checks.")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: ./watchdog [flags] workers.txt")
		flag.PrintDefaults()
		return
	}
	workersFile := flag.Arg(0)

	workers, err := readWorkers(workersFile)
	if err != nil {
		log.Fatalf("Failed to read workers: %v", err)
	}

	status := make(map[string]bool) // true=online, false=offline
	var mu sync.Mutex               // protects status map

	for {
		var wg sync.WaitGroup
		for name, addr := range workers {
			wg.Add(1)
			go func(name, addr string) {
				defer wg.Done()
				online := isOnline(addr, port, time.Duration(timeoutSec)*time.Second)

				mu.Lock()
				prev, known := status[name]
				if !known {
					status[name] = online
					if online {
						log.Printf("[STARTUP] %s (%s) is ONLINE\n", name, addr)
					} else {
						log.Printf("[STARTUP] %s (%s) is OFFLINE\n", name, addr)
					}
				} else if online != prev {
					status[name] = online
					if online {
						log.Printf("[RECOVERY] %s (%s) is back ONLINE\n", name, addr)
					} else {
						log.Printf("[DOWN] %s (%s) went OFFLINE\n", name, addr)
						alertWorkerDown(name, addr) // call alert stub here
					}
				}
				mu.Unlock()
			}(name, addr)
		}
		wg.Wait()
		time.Sleep(time.Duration(checkInterval) * time.Second)
	}
}
