package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image/png"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"file-transfer/backend/config"
	"file-transfer/backend/handler"
	"file-transfer/backend/middleware"
	"file-transfer/backend/service"
	"file-transfer/pkg/xfer/stun"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
)

const banner = `
 ██████╗ ██████╗  ██████╗ ███████╗███████╗████████╗██████╗  █████╗ ███╗   ██╗███████╗███████╗██████╗
 ██╔══██╗██╔══██╗██╔═══██╗██╔════╝██╔════╝╚══██╔══╝██╔══██╗██╔══██╗████╗  ██║██╔════╝██╔════╝██╔══██╗
 ██████╔╝██████╔╝██║   ██║███████╗█████╗     ██║   ██████╔╝███████║██╔██╗ ██║███████╗█████╗  ██████╔╝
 ██╔══██╗██╔══██╗██║   ██║╚════██║██╔══╝     ██║   ██╔══██╗██╔══██║██║╚██╗██║╚════██║██╔══╝  ██╔══██╗
 ██████╔╝██║  ██║╚██████╔╝███████║███████╗   ██║   ██║  ██║██║  ██║██║ ╚████║███████║███████╗██║  ██║
 ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝╚═╝  ╚═╝
`

type Role string

const (
	RoleSender    Role = "sender"
	RoleReceiver  Role = "receiver"
	RoleBoth      Role = "both"
)

type NetworkInfo struct {
	Type       string
	Address    string
	IsIPv6     bool
	IsPublic   bool
	Accessible string
}

func main() {
	cfg := config.ParseConfig()

	fmt.Println(banner)
	fmt.Println("文件传输系统 - 支持跨网段传输")
	fmt.Println()

	role := detectRole(cfg)
	fmt.Printf("角色模式: %s\n", getRoleDescription(role))
	fmt.Printf("� 共享目录: %s\n", cfg.ShareDir)
	fmt.Printf("� 输出目录: %s\n", cfg.OutputDir)
	fmt.Printf("� 加密: %s\n", boolToYesNo(cfg.Encrypt))

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(middleware.CORS())

	fileService := service.NewFileService(cfg.ShareDir, cfg.OutputDir)
	fileHandler := handler.NewFileHandler(fileService, cfg.ReceiveOnly)
	fileHandler.RegisterRoutes(r)

	cryptoService := service.NewCryptoService()
	cryptoHandler := handler.NewCryptoHandler(cryptoService)
	cryptoHandler.RegisterRoutes(r)

	qrcodeHandler := handler.NewQRCodeHandler()
	qrcodeHandler.RegisterRoutes(r)

	p2pHandler := handler.NewP2PHandler()
	p2pHandler.RegisterRoutes(r)

	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	frontendPath := filepath.Join(exeDir, "frontend")

	if _, err := os.Stat(frontendPath); os.IsNotExist(err) {
		frontendPath = filepath.Join(exeDir, "..", "frontend")
	}
	frontendPath, _ = filepath.Abs(frontendPath)

	if _, err := os.Stat(frontendPath); os.IsNotExist(err) {
		frontendPath = "./frontend"
	}

	r.StaticFile("/", filepath.Join(frontendPath, "index.html"))
	r.Static("/css", filepath.Join(frontendPath, "css"))
	r.Static("/js", filepath.Join(frontendPath, "js"))

	r.NoRoute(func(c *gin.Context) {
		c.File(filepath.Join(frontendPath, "index.html"))
	})

	networkInfo := collectNetworkInfo(cfg.Port, cfg.Encrypt, cryptoService)

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("🌐 可访问地址:")
	fmt.Println("=" + strings.Repeat("=", 60))

	for _, info := range networkInfo {
		fmt.Printf("[%s] %s\n", info.Type, info.Accessible)
	}

	fmt.Println()
	fmt.Println("📱 扫描二维码访问:")
	fmt.Println()

	for _, info := range networkInfo {
		url := info.Accessible
		if strings.Contains(url, "#key=") {
			parts := strings.Split(url, "#key=")
			url = parts[0]
		}

		asciiQR, err := generateASCIIQRCode(url)
		if err != nil {
			fmt.Printf("  [%s] %s\n", info.Type, info.Accessible)
		} else {
			fmt.Printf("  [%s]\n%s\n", info.Type, asciiQR)
		}
		fmt.Printf("  链接: %s\n\n", info.Accessible)
	}

	fmt.Println()
	fmt.Printf("🔗 主要访问: %s\n", networkInfo[0].Accessible)
	fmt.Println()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 Server starting on :%s\n", cfg.Port)
		if err := r.Run(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌  Server failed to start: %v", err)
		}
	}()

	if role == RoleSender || role == RoleBoth {
		go runSenderMode(cfg)
	}
	if role == RoleReceiver || role == RoleBoth {
		go runReceiverMode(cfg)
	}

	<-sigChan
	fmt.Println("\n👋 Shutting down server...")
}

func detectRole(cfg *config.Config) Role {
	if cfg.ReceiveOnly {
		return RoleReceiver
	}

	args := os.Args
	if len(args) > 1 {
		switch strings.ToLower(args[len(args)-1]) {
		case "sender", "-sender", "--sender":
			return RoleSender
		case "receiver", "-receiver", "--receiver":
			return RoleReceiver
		case "both", "-both", "--both":
			return RoleBoth
		}
	}

	if runtime.GOOS == "windows" {
		fmt.Println("选择运行模式:")
		fmt.Println("  1. 发送方 (sender) - 分享文件给其他人")
		fmt.Println("  2. 接收方 (receiver) - 接收他人文件")
		fmt.Println("  3. 两者 (both) - 同时支持发送和接收")
		fmt.Println()
		fmt.Print("请输入选择 (1/2/3，默认1): ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "2":
			return RoleReceiver
		case "3":
			return RoleBoth
		default:
			return RoleSender
		}
	}

	return RoleSender
}

func getRoleDescription(role Role) string {
	switch role {
	case RoleSender:
		return "发送方 (分享文件)"
	case RoleReceiver:
		return "接收方 (接收文件)"
	case RoleBoth:
		return "两者 (发送和接收)"
	default:
		return "未知"
	}
}

func collectNetworkInfo(port string, encrypt bool, cryptoService *service.CryptoService) []NetworkInfo {
	var infoList []NetworkInfo

	ifaces, _ := net.Interfaces()
	var localAddrs []net.Addr

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		localAddrs = append(localAddrs, addrs...)
	}

	var ipv4Addrs, ipv6Addrs []string
	for _, addr := range localAddrs {
		ip := extractIP(addr)
		if ip == nil {
			continue
		}
		if ip.To4() != nil {
			ipv4Addrs = append(ipv4Addrs, ip.String())
		} else {
			ipv6Addrs = append(ipv6Addrs, ip.String())
		}
	}

	for _, ip := range ipv6Addrs {
		infoList = append(infoList, NetworkInfo{
			Type:       "IPv6",
			Address:    ip,
			IsIPv6:     true,
			IsPublic:   false,
			Accessible: fmt.Sprintf("http://[%s]:%s", ip, port),
		})
	}

	for _, ip := range ipv4Addrs {
		infoList = append(infoList, NetworkInfo{
			Type:       "LAN",
			Address:    ip,
			IsIPv6:     false,
			IsPublic:   false,
			Accessible: fmt.Sprintf("http://%s:%s", ip, port),
		})
	}

	go func() {
		stunClient := stun.NewClient(nil)
		if addr, err := stunClient.GetPublicAddr(); err == nil {
			publicInfo := NetworkInfo{
				Type:       "STUN",
				Address:    addr,
				IsIPv6:     false,
				IsPublic:   true,
				Accessible: fmt.Sprintf("http://%s:%s", addr, port),
			}
			infoList = append(infoList, publicInfo)
		}
	}()

	time.Sleep(1 * time.Second)

	key := ""
	if encrypt {
		key, _ = cryptoService.GenerateKey()
	}

	for i := range infoList {
		if key != "" {
			infoList[i].Accessible = fmt.Sprintf("%s/#key=%s", infoList[i].Accessible, key)
		}
	}

	if len(infoList) == 0 {
		infoList = append(infoList, NetworkInfo{
			Type:       "Local",
			Address:    "127.0.0.1",
			IsIPv6:     false,
			IsPublic:   false,
			Accessible: fmt.Sprintf("http://127.0.0.1:%s", port),
		})
	}

	return infoList
}

func extractIP(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	}
	return nil
}

func runSenderMode(cfg *config.Config) {
	fmt.Println("[发送方模式] 等待接收方连接...")
	fmt.Println("提示: 分享您的地址给接收方设备")
}

func runReceiverMode(cfg *config.Config) {
	fmt.Println("[接收方模式] 等待发送方发送文件...")
	fmt.Println("提示: 扫描发送方的二维码建立连接")
}

func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func generateASCIIQRCode(url string) (string, error) {
	pngData, err := qrcode.Encode(url, qrcode.Medium, 20)
	if err != nil {
		return "", err
	}

	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	var result strings.Builder

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			brightness := (r + g + b) / 3

			r2, g2, b2, _ := img.At(x, y+1).RGBA()
			brightness2 := (r2 + g2 + b2) / 3

			if brightness > 0x8000 && brightness2 > 0x8000 {
				result.WriteString("  ")
			} else if brightness <= 0x8000 && brightness2 <= 0x8000 {
				result.WriteString("██")
			} else if brightness <= 0x8000 && brightness2 > 0x8000 {
				result.WriteString("▀▀")
			} else {
				result.WriteString("▄▄")
			}
		}
		result.WriteString("\n")
	}

	return result.String(), nil
}

func init() {
	if _, err := os.Stat("vendor"); os.IsNotExist(err) {
	} else {
	}
}