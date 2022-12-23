package main

import (
	"bufio"
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"strings"
	"time"

	"github.com/AlexwellChen/chord/utils"
)

func HandleConnection(listener net.Listener, node *utils.Node) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept failed:", err.Error())
			continue
		}
		go jsonrpc.ServeConn(conn)
	}
}

func StartChord(args utils.Arguments) *utils.Node {
	// Check if the command line arguments are valid
	valid := utils.CheckArgsValid(args)
	var node *utils.Node
	if valid == -1 {
		fmt.Println("Invalid command line arguments")
		os.Exit(1)
	} else {
		fmt.Println("Valid command line arguments")
		// Create new Node
		node = utils.NewNode(args)

		IPAddr := fmt.Sprintf("%s:%d", args.Address, args.Port)
		tcpAddr, err := net.ResolveTCPAddr("tcp4", IPAddr)
		if err != nil {
			fmt.Println("ResolveTCPAddr failed:", err.Error())
			os.Exit(1)
		}
		rpc.Register(node)

		listener, err := net.Listen("tcp", tcpAddr.String())
		if err != nil {
			fmt.Println("ListenTCP failed:", err.Error())
			os.Exit(1)
		}
		fmt.Println("Local node listening on ", tcpAddr)
		// Use a separate goroutine to accept connection
		go HandleConnection(listener, node)

		if valid == 0 {
			// Join exsiting chord
			RemoteAddr := fmt.Sprintf("%s:%d", args.JoinAddress, args.JoinPort)

			// Connect to the remote node
			fmt.Println("Connecting to the remote node..." + RemoteAddr)
			err := node.JoinChord(utils.NodeAddress(RemoteAddr))
			if err != nil {
				fmt.Println("Join RPC call failed")
				os.Exit(1)
			} else {
				fmt.Println("Join RPC call success")
			}
		} else if valid == 1 {
			// Create new chord
			node.CreateChord()
			// Combine address and port, convert port to string
		}

		// Start periodic tasks
		Se_stab := utils.ScheduledExecutor{Delay: time.Duration(args.Stabilize) * time.Millisecond, Quit: make(chan int)}
		Se_stab.Start(func() {
			node.Stablize()
		})

		Se_ff := utils.ScheduledExecutor{Delay: time.Duration(args.FixFingers) * time.Millisecond, Quit: make(chan int)}
		Se_ff.Start(func() {
			node.FixFingers()
		})

		Se_cp := utils.ScheduledExecutor{Delay: time.Duration(args.CheckPred) * time.Millisecond, Quit: make(chan int)}
		Se_cp.Start(func() {
			node.CheckPredecessor()
		})

		node.Se_cp = &Se_cp
		node.Se_ff = &Se_ff
		node.Se_stab = &Se_stab
	}
	return node
}

func main() {
	// Parse command line arguments
	Arguments := utils.GetCmdArgs()
	fmt.Println(Arguments)
	node := StartChord(Arguments)
	// Get user input for printing states
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter command: ")
		command, _ := reader.ReadString('\n')
		command = strings.TrimSpace(command)
		command = strings.ToUpper(command)
		if command == "PRINTSTATE" || command == "PS" {
			node.PrintState()
			utils.GetLocalAddress()
		} else if command == "LOOKUP" || command == "L" {
			fmt.Println("Please enter the key you want to lookup")
			key, _ := reader.ReadString('\n')
			key = strings.TrimSpace(key)
			fmt.Println(key)
			resultAddr, err := utils.ClientLookUp(key, node)
			if err != nil {
				fmt.Print(err)
			} else {
				fmt.Println("The address of the key is ", resultAddr)

			}
			// Check if the key is stored in the node
			checkFileExistRPCReply := utils.CheckFileExistRPCReply{}
			err = utils.ChordCall(resultAddr, "Node.CheckFileExistRPC", key, &checkFileExistRPCReply)
			if err != nil {
				fmt.Println("Check file exist RPC call failed")
			} else {
				if checkFileExistRPCReply.Exist {
					// Get the address of the node that stores the file
					var getNameRPCReply utils.GetNameRPCReply
					err = utils.ChordCall(resultAddr, "Node.GetNameRPC", "", &getNameRPCReply)
					if err != nil {
						fmt.Println("Get name RPC call failed")
					} else {
						fmt.Println("The file is stored at ", getNameRPCReply.Name)
					}
				} else {
					fmt.Println("The file is not stored in the node")
				}
			}
		} else if command == "STOREFILE" || command == "S" {
			fmt.Println("Please enter the file name you want to store")
			fileName, _ := reader.ReadString('\n')
			fileName = strings.TrimSpace(fileName)
			err := utils.ClientStoreFile(fileName, node)
			if err != nil {
				fmt.Print(err)
			} else {
				fmt.Println("Store file success")
			}
		} else if command == "QUIT" || command == "Q" {
			// Quit the program
			// Assign a value to quit channel to stop periodic tasks
			node.Quit()
			os.Exit(0)
		} else if command == "GET" || command == "G" {
			// Get file from the network
			fmt.Println("Please enter the file name you want to get")
			fileName, _ := reader.ReadString('\n')
			fileName = strings.TrimSpace(fileName)
			err := utils.ClientGetFile(fileName, node)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Get file success")
			}
		} else {
			fmt.Println("Invalid command")
		}
	}
}
