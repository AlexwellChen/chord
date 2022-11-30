package main

import (
	"bufio"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strings"
	"time"
)

type ScheduledExecutor struct {
	delay  time.Duration
	ticker time.Ticker
	quit   chan int
}

// Use Go channel to implement periodic tasks
func (se *ScheduledExecutor) Start(task func()) {
	se.ticker = *time.NewTicker(se.delay)
	go func() {
		for {
			select {
			case <-se.ticker.C:
				// Use goroutine to run the task to avoid blocking user input
				go task()
			case <-se.quit:
				se.ticker.Stop()
				return
			}
		}
	}()
}

func HandleConnection(listener *net.TCPListener, node *Node) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept failed:", err.Error())
			continue
		}
		rpc.ServeConn(conn)
	}
}
func main() {
	// Parse command line arguments
	Arguments := getCmdArgs()
	fmt.Println(Arguments)
	// Check if the command line arguments are valid
	valid := CheckArgsValid(Arguments)
	if valid == -1 {
		fmt.Println("Invalid command line arguments")
		os.Exit(1)
	} else {
		fmt.Println("Valid command line arguments")
		// Create new Node
		node := NewNode(Arguments)
		if valid == 0 {
			// Join exsiting chord

			RemoteAddr := fmt.Sprintf("%s:%d", Arguments.JoinAddress, Arguments.JoinPort)
			// Connect to the remote node
			// TODO: Use ChordCall function instead
			node.joinChord(NodeAddress(RemoteAddr))
			fmt.Println("Join RPC call success")
		} else if valid == 1 {
			// Create new chord
			node.createChord()
			// Combine address and port, convert port to string
			IPAddr := fmt.Sprintf("%s:%d", Arguments.Address, Arguments.Port)
			tcpAddr, err := net.ResolveTCPAddr("tcp4", IPAddr)
			if err != nil {
				fmt.Println("ResolveTCPAddr failed:", err.Error())
				os.Exit(1)
			}
			// Listen to the address
			listener, err := net.ListenTCP("tcp", tcpAddr)
			if err != nil {
				fmt.Println("ListenTCP failed:", err.Error())
				os.Exit(1)
			}
			fmt.Println("Created chord ring on ", IPAddr)
			// Use a separate goroutine to accept connection
			go HandleConnection(listener, node)
		}

		// Start periodic tasks
		se := ScheduledExecutor{delay: time.Duration(Arguments.Stabilize) * time.Millisecond, quit: make(chan int)}
		se.Start(func() {
			// node.stabilize()
		})
		// TODO: Check if this usage of starting periodic task is correct, do similar things for other periodic tasks

		// Get user input for printing states
		reader := bufio.NewReader(os.Stdin)
		for {
			command, _ := reader.ReadString('\n')
			command = strings.TrimSpace(command)
			command = strings.ToUpper(command)
			if command == "PRINTSTATE" || command == "PS" {
				node.printState()
			} else if command == "LOOKUP" || command == "L" {
				fmt.Println("Please enter the key you want to lookup")
				key, _ := reader.ReadString('\n')
				key = strings.TrimSpace(key)
				fmt.Println(key)
				// TODO: Implement lookup function
				// node.lookUp(key)
			} else if command == "STOREFILE" || command == "S" {
				fmt.Println("Please enter the file name you want to store")
				fileName, _ := reader.ReadString('\n')
				fileName = strings.TrimSpace(fileName)
				fmt.Println(fileName)
				// TODO: Implement store file function
				// node.storeFile(fileName)
			} else if command == "QUIT" || command == "Q" {
				// Quit the program
				// Assign a value to quit channel to stop periodic tasks
				se.quit <- 1
				os.Exit(0)
			} else {
				fmt.Println("Invalid command")
			}
		}
	}

}