package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

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
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// 请求目标 URL 并获取响应数据
			client := http.Client{}
			req, err := http.NewRequest(http.MethodGet, remoteUrl, nil)
			if err != nil {
				http.Error(w, "Error building remote request: "+err.Error(), http.StatusInternalServerError)
				fmt.Println("构造请求失败：", err)
				return
			}
			// 添加请求头
			// req.Header.Add("Content-type", "application/json;charset=utf-8")
			req.Header.Add("MetadataVersion", strconv.FormatUint(GetCompressProcessor().metadata.Version, 10))
			resp, err := client.Do(req)
			// resp, err := http.Get(remoteUrl)
			if err != nil {
				http.Error(w, "Error getting remote metrics: "+err.Error(), http.StatusInternalServerError)
				fmt.Println("请求失败：", err)
				return
			}
			defer resp.Body.Close()

			// 读取响应数据并处理
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, "Error reading remote data: "+err.Error(), http.StatusInternalServerError)
				fmt.Println("读取响应失败：", err)
				return
			}

			// fmt.Println(string(data))
			processedData, err := GetCompressProcessor().process(data)
			if err != nil {
				http.Error(w, "Error processing remote data: "+err.Error(), http.StatusInternalServerError)
				fmt.Println("解析响应失败：", err)
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
