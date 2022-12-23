package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/AlexwellChen/chord/utils"
)

func main() {
	// Parse command line arguments
	Arguments := utils.GetCmdArgs()
	fmt.Println(Arguments)
	node := utils.StartChord(Arguments)
	// Get user input for printing states
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter command: ")
		command, _ := reader.ReadString('\n')
		command = strings.TrimSpace(command)
		command = strings.ToUpper(command)
		if command == "PRINTSTATE" || command == "PS" {
			node.PrintState()
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
