package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
)

var pool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 1024) // 初始化一个大小为0，容量为1024的字节切片
	},
}

func handleFileMode(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("读取文件失败：", err)
		return
	}
	processedData, err := GetCompressProcessor().process(data)
	if err != nil {
		fmt.Println("处理文件失败：", err)
		return
	}
	fmt.Println(string(processedData))
}

func handleHTTPMode(remoteUrl string, localPort int) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// 请求目标 URL 并获取响应数据
			// TODO 加上 header
			resp, err := http.Get(remoteUrl)
			if err != nil {
				fmt.Println("请求失败：", err)
				return
			}
			defer resp.Body.Close()

			// 读取响应数据并处理
			// data, err := io.ReadAll(resp.Body)
			// if err != nil {
			// 	fmt.Println("读取响应失败：", err)
			// 	return
			// }

			data := pool.Get().([]byte) // 从池中获取[]byte
			defer pool.Put(data)        // 将[]byte放回池中以便重用

			data = data[:0] // 重置data，确保其长度为0

			_, err = io.ReadFull(resp.Body, data)
			if err != nil {
				http.Error(w, "Error reading remote data: "+err.Error(), http.StatusInternalServerError)
				return
			}

			processedData, err := GetCompressProcessor().process(data)
			if err != nil {
				http.Error(w, "Error processing remote data: "+err.Error(), http.StatusInternalServerError)
				return
			}

			w.Write(processedData)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("HTTP server listening on", localPort)
	err := http.ListenAndServe(":"+strconv.Itoa(localPort), nil)
	if err != nil {
		fmt.Println("HTTP server failed to start:", err)
	}
}

func main() {

	var filename string
	var remoteURL string
	var localPort int

	flag.StringVar(&filename, "f", "", "文件名")
	flag.StringVar(&remoteURL, "s", "", "远程 IP 地址：端口")
	flag.IntVar(&localPort, "p", -1, "转发到本地端口")

	flag.Parse()

	if filename != "" {
		handleFileMode(filename)
	} else if remoteURL != "" && localPort != -1 {
		// 启动 HTTP handler
		handleHTTPMode(remoteURL, localPort)
	} else {
		fmt.Println("Usage: ./main [-f filename] [-s url -p port]")
	}
}
