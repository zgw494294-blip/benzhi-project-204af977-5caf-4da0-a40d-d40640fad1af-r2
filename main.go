package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/httpapi"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func main() {
	selfcheck := flag.Bool("selfcheck", false, "运行完整业务自检并退出")
	address := flag.String("addr", ":8080", "HTTP 服务监听地址")
	ledgerPath := flag.String("ledger", "data/ledger.json", "本地账本路径")
	flag.Parse()
	if *selfcheck {
		if err := httpapi.RunSelfCheck(); err != nil {
			fmt.Fprintf(os.Stderr, "自检失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("自检通过：创建、测量、幂等重试、版本冲突、返工、复核、封存、证书和审计查询链路均正常")
		return
	}
	repository, err := store.New(*ledgerPath)
	if err != nil {
		log.Fatalf("加载本地账本失败: %v", err)
	}
	service := calibration.NewService(repository)
	server := &http.Server{Addr: *address, Handler: httpapi.NewServer(service)}
	displayAddress := *address
	if strings.HasPrefix(displayAddress, ":") {
		displayAddress = "localhost" + displayAddress
	}
	log.Printf("星测校准台已启动: http://%s", displayAddress)
	log.Printf("账本路径: %s", repository.Path())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP 服务停止: %v", err)
	}
}
