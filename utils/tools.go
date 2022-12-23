package utils

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"regexp"
	"strings"
	"time"
)

/*------------------------------------------------------------*/
/*                  Comm Interface By: Alexwell               */
/*------------------------------------------------------------*/

/*
* @description: Communication interface between nodes
* @param: 		targetNode: the address of the node to be connected
* @param: 		method: the name of the method to be called, e.g. "Node.FindSuccessorRPC".
*						method need to be registered in the RPC server, and have Golang compliant RPC method style
* @param:		request: the request to be sent
* @param:		reply: the reply to be received
* @return:		error: the error returned by the RPC call
 */
func ChordCall(targetNode NodeAddress, method string, request interface{}, reply interface{}) error {
	if len(strings.Split(string(targetNode), ":")) != 2 {
		fmt.Println("Error: targetNode address is not in the correct format: ", targetNode)
		return errors.New("Error: targetNode address is not in the correct format: " + string(targetNode))
	}
	ip := strings.Split(string(targetNode), ":")[0]
	port := strings.Split(string(targetNode), ":")[1]

	targetNodeAddr := ip + ":" + port
	// conn, err := tls.Dial("tcp", targetNodeAddr, &tls.Config{InsecureSkipVerify: true})
	// client := jsonrpc.NewClient(conn)
	client, err := jsonrpc.Dial("tcp", targetNodeAddr)
	if err != nil {
		fmt.Println("Method: ", method, "Dial Error: ", err)
		return err
	}
	defer client.Close()
	err = client.Call(method, request, reply)
	if err != nil {
		fmt.Println("Call Error:", err)
		return err
	}
	return nil
}

/*------------------------------------------------------------*/
/*                     Tool Functions Below                   */
/*------------------------------------------------------------*/

type Arguments struct {
	// Read command line arguments
	Address     NodeAddress // Current node address
	Port        int         // Current node port
	JoinAddress NodeAddress // Joining node address
	JoinPort    int         // Joining node port
	Stabilize   int         // The time in milliseconds between invocations of stabilize.
	FixFingers  int         // The time in milliseconds between invocations of fix_fingers.
	CheckPred   int         // The time in milliseconds between invocations of check_predecessor.
	Successors  int
	ClientName  string
}

func GetCmdArgs() Arguments {
	// Read command line arguments
	var a string  // Current node address
	var p int     // Current node port
	var ja string // Joining node address
	var jp int    // Joining node port
	var ts int    // The time in milliseconds between invocations of stabilize.
	var tff int   // The time in milliseconds between invocations of fix_fingers.
	var tcp int   // The time in milliseconds between invocations of check_predecessor.
	var r int     // The number of successors to maintain.
	var i string  // Client name

	// Parse command line arguments
	flag.StringVar(&a, "a", "localhost", "Current node address")
	flag.IntVar(&p, "p", 8000, "Current node port")
	flag.StringVar(&ja, "ja", "Unspecified", "Joining node address")
	flag.IntVar(&jp, "jp", 8000, "Joining node port")
	flag.IntVar(&ts, "ts", 3000, "The time in milliseconds between invocations of stabilize.")
	flag.IntVar(&tff, "tff", 1000, "The time in milliseconds between invocations of fix_fingers.")
	flag.IntVar(&tcp, "tcp", 3000, "The time in milliseconds between invocations of check_predecessor.")
	flag.IntVar(&r, "r", 3, "The number of successors to maintain.")
	flag.StringVar(&i, "i", "Default", "Client ID/Name")
	flag.Parse()

	// Return command line arguments
	return Arguments{
		Address:     NodeAddress(a),
		Port:        p,
		JoinAddress: NodeAddress(ja),
		JoinPort:    jp,
		Stabilize:   ts,
		FixFingers:  tff,
		CheckPred:   tcp,
		Successors:  r,
		ClientName:  i,
	}
}

// Use Go channel to implement periodic tasks
func (se *ScheduledExecutor) Start(task func()) {
	se.Ticker = *time.NewTicker(se.Delay)
	go func() {
		for {
			select {
			case <-se.Ticker.C:
				// Use goroutine to run the task to avoid blocking user input
				go task()
			case <-se.Quit:
				se.Ticker.Stop()
				return
			}
		}
	}()
}

func CheckArgsValid(args Arguments) int {
	// Check if Ip address is valid or not
	if net.ParseIP(string(args.Address)) == nil && args.Address != "localhost" {
		fmt.Println("IP address is invalid")
		return -1
	}
	// Check if port is valid
	if args.Port < 1024 || args.Port > 65535 {
		fmt.Println("Port number is invalid")
		return -1
	}

	// Check if durations are valid
	if args.Stabilize < 1 || args.Stabilize > 60000 {
		fmt.Println("Stabilize time is invalid")
		return -1
	}
	if args.FixFingers < 1 || args.FixFingers > 60000 {
		fmt.Println("FixFingers time is invalid")
		return -1
	}
	if args.CheckPred < 1 || args.CheckPred > 60000 {
		fmt.Println("CheckPred time is invalid")
		return -1
	}

	// Check if number of successors is valid
	if args.Successors < 1 || args.Successors > 32 {
		fmt.Println("Successors number is invalid")
		return -1
	}

	// Check if client name is s a valid string matching the regular expression [0-9a-fA-F]{40}
	if args.ClientName != "Default" {
		matched, err := regexp.MatchString("[0-9a-fA-F]*", args.ClientName)
		if err != nil || !matched {
			fmt.Println("Client Name is invalid")
			return -1
		}
	}

	// Check if joining address and port is valid or not
	if args.JoinAddress != "Unspecified" {
		// Addr is specified, check if addr & port are valid
		if net.ParseIP(string(args.JoinAddress)) != nil || args.JoinAddress == "localhost" {
			// Check if join port is valid
			if args.JoinPort < 1024 || args.JoinPort > 65535 {
				fmt.Println("Join port number is invalid")
				return -1
			}
			// Join the chord
			return 0
		} else {
			fmt.Println("Joining address is invalid")
			return -1
		}
	} else {
		// Join address is not specified, create a new chord ring
		// ignroe jp input
		return 1
	}
}

func StrHash(elt string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(elt))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}

func between(start, elt, end *big.Int, inclusive bool) bool {
	if end.Cmp(start) > 0 { // start < end
		return (start.Cmp(elt) < 0 && elt.Cmp(end) < 0) || (inclusive && elt.Cmp(end) == 0)
	} else {
		return start.Cmp(elt) < 0 || elt.Cmp(end) < 0 || (inclusive && elt.Cmp(end) == 0)
	}
}

func ClientLookUp(key string, node *Node) (NodeAddress, error) {
	// Find the successor of key
	// Return the successor's address and port
	newKey := StrHash(key) // Use file name as key
	addr := find(newKey, node.Address)

	if addr == "-1" {
		return "", errors.New("cannot find the store position of the key")
	} else {
		return addr, nil
	}
}

// File structure
type FileRPC struct {
	Id      *big.Int
	Name    string
	Content []byte
}

func ClientStoreFile(fileName string, node *Node) error {
	// Store the file in the node
	// Return the address and port of the node that stores the file
	addr, err := ClientLookUp(fileName, node)
	if err != nil {
		return err
	} else {
		fmt.Println("The file is stored in node: ", addr)
	}
	// Open file and pack into fileRPC
	currentNodeFileUploadPath := "../files/" + node.Name + "/file_upload/"
	filepath := currentNodeFileUploadPath + fileName
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("Cannot open the file")
		return err
	}
	defer file.Close()
	// Init new file struct and put content into it
	newFile := FileRPC{}
	newFile.Name = fileName
	newFile.Content, err = ioutil.ReadAll(file)
	newFile.Id = StrHash(fileName)
	newFile.Id.Mod(newFile.Id, hashMod)
	if err != nil {
		return err
	} else {
		// Encrypted file content
		if node.EncryptFlag {
			newFile.Content = node.encryptFile(newFile.Content)
		}
		reply := new(StoreFileRPCReply)
		reply.Backup = false
		err = ChordCall(addr, "Node.StoreFileRPC", newFile, &reply)
		if reply.Err != nil && err != nil {
			return errors.New("cannot store the file")
		}
	}
	return nil
}

func ClientGetFile(fileName string, node *Node) error {
	// Get the file from the node
	addr, err := ClientLookUp(fileName, node)
	if err != nil {
		return err
	} else {
		fmt.Println("The file is stored in node: ", addr)
	}
	// Open file and pack into fileRPC
	// currentNodeFileDownloadPath := "../files/" + node.Name + "/file_download/"
	// filepath := currentNodeFileDownloadPath + fileName
	file := FileRPC{}
	file.Name = fileName
	file.Id = StrHash(fileName)
	file.Id.Mod(file.Id, hashMod)
	err = ChordCall(addr, "Node.GetFileRPC", file, &file)
	if err != nil {
		fmt.Println("Cannot get the file")
		return err
	} else {
		// Decrypt file content
		if node.EncryptFlag {
			file.Content = node.decryptFile(file.Content)
		}
		// Write file to local
		success := node.storeLocalFile(file)
		if !success {
			return errors.New("cannot store local file")
		}

	}
	return nil
}

func GetLocalAddress() string {
	// Obtain the local ip address from dns server 8.8.8:80
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

type IP struct {
	Query string
}

func Getip2() string {
	req, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		return err.Error()
	}
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err.Error()
	}

	var ip IP
	fmt.Println("body: ", string(body))
	json.Unmarshal(body, &ip)

	return ip.Query
}

func HandleConnection(listener net.Listener, node *Node) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept failed:", err.Error())
			continue
		}
		go jsonrpc.ServeConn(conn)
	}
}

func StartChord(args Arguments) *Node {
	// Check if the command line arguments are valid
	valid := CheckArgsValid(args)
	var node *Node
	if valid == -1 {
		fmt.Println("Invalid command line arguments")
		os.Exit(1)
	} else {
		fmt.Println("Valid command line arguments")
		// Create new Node
		node = NewNode(args)

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
			err := node.JoinChord(NodeAddress(RemoteAddr))
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
		Se_stab := ScheduledExecutor{Delay: time.Duration(args.Stabilize) * time.Millisecond, Quit: make(chan int)}
		Se_stab.Start(func() {
			node.Stablize()
		})

		Se_ff := ScheduledExecutor{Delay: time.Duration(args.FixFingers) * time.Millisecond, Quit: make(chan int)}
		Se_ff.Start(func() {
			node.FixFingers()
		})

		Se_cp := ScheduledExecutor{Delay: time.Duration(args.CheckPred) * time.Millisecond, Quit: make(chan int)}
		Se_cp.Start(func() {
			node.CheckPredecessor()
		})

		node.Se_cp = &Se_cp
		node.Se_ff = &Se_ff
		node.Se_stab = &Se_stab
	}
	return node
}
