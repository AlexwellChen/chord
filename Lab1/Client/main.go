package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
)

func main() {
	server := "localhost:8080"
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)

	if err != nil {
		fmt.Println(os.Stderr, "Fatal error: ", err)
		os.Exit(1)
	}

	//建立服务器连接
	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	if err != nil {
		fmt.Println(conn.RemoteAddr().String(), os.Stderr, "Fatal error:", err)
		os.Exit(1)
	}

	fmt.Println("connection success")
	sender(conn)
	fmt.Println("send over")

}

func sender(conn *net.TCPConn) {
	host_addr := conn.RemoteAddr().String()

	url := "http://" + host_addr + "/root/1.txt"
	fmt.Println("url:", url)
	request, _ := http.NewRequest("GET", url, nil)
	err := request.Write(conn)
	// fmt.Println("Request: ", request)
	// src := request.URL.String()             // Todo: 把request转换成字符串
	// msgBack, err := conn.Write([]byte(src)) //给服务器发信息

	if err != nil {
		fmt.Println(conn.RemoteAddr().String(), " Error: ", err)
		os.Exit(1)
	}
	buffer := make([]byte, 1024)
	msg, _ := conn.Read(buffer) //接受服务器信息
	fmt.Println(conn.RemoteAddr().String(), "服务器返回: ", string(buffer[:msg]))
	// // conn.Write([]byte("ok")) //在告诉服务器，它的反馈收到了。
}