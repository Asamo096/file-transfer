package config

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	Port       string
	ShareDir   string
	OutputDir  string
	Encrypt    bool
	ReceiveOnly bool
}

func ParseConfig() *Config {
	cfg := &Config{
		Port:      "8888",
		ShareDir:  ".",
		OutputDir: ".",
		Encrypt:   false,
		ReceiveOnly: false,
	}

	flag.StringVar(&cfg.Port, "p", "8888", "端口号")
	flag.StringVar(&cfg.Port, "port", "8888", "端口号")
	flag.StringVar(&cfg.ShareDir, "d", ".", "共享/输出目录")
	flag.StringVar(&cfg.ShareDir, "dir", ".", "共享/输出目录")
	flag.BoolVar(&cfg.Encrypt, "e", false, "启用加密")
	flag.BoolVar(&cfg.Encrypt, "encrypt", false, "启用加密")
	flag.BoolVar(&cfg.ReceiveOnly, "r", false, "仅接收模式")
	flag.BoolVar(&cfg.ReceiveOnly, "receive", false, "仅接收模式")

	flag.Usage = func() {
		PrintHelp()
	}

	flag.Parse()

	cfg.OutputDir = cfg.ShareDir

	if flag.NArg() > 0 {
		PrintHelp()
		os.Exit(1)
	}

	return cfg
}

func PrintHelp() {
	fmt.Println("用法: file-transfer [选项] [角色]")
	fmt.Println("\n角色 (可选):")
	fmt.Println("  sender    发送方模式 - 分享文件给其他设备")
	fmt.Println("  receiver  接收方模式 - 接收他人文件")
	fmt.Println("  both      两者模式 - 同时支持发送和接收 (默认)")
	fmt.Println("\n选项:")
	fmt.Println("  -p, --port <端口>     指定服务端口 (默认: 8888)")
	fmt.Println("  -d, --dir <目录>      指定共享/输出目录 (默认: 当前目录)")
	fmt.Println("  -e, --encrypt         启用端到端加密")
	fmt.Println("  -r, --receive         仅接收模式 (等同于 receiver)")
	fmt.Println("  -h, --help            显示帮助信息")
	fmt.Println("\n示例:")
	fmt.Println("  file-transfer                    # 启动，两者模式")
	fmt.Println("  file-transfer sender            # 发送方模式")
	fmt.Println("  file-transfer receiver           # 接收方模式")
	fmt.Println("  file-transfer -p 8080 -d /path  # 指定端口和目录")
}
