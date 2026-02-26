package main

import (
	"fmt"
	"os"
)

// 修复后的代码
func main() {
	// 修复1: 使用环境变量代替硬编码
	password := os.Getenv("APP_PASSWORD")
	token := os.Getenv("APP_TOKEN")
	
	if password == "" {
		password = "default-password"
	}
	if token == "" {
		token = "default-token"
	}
	
	// 修复2: 正确处理错误
	file, err := os.Open("config.txt")
	if err != nil {
		fmt.Printf("警告: 无法打开配置文件: %v\n", err)
		// 继续执行，不因为配置文件缺失而失败
	} else {
		defer file.Close()
	}
	
	// 修复3: 避免 nil 指针
	var data *string
	sample := "sample data"
	data = &sample // 初始化指针
	if data != nil {
		fmt.Println(*data)
	}
	
	fmt.Println("Password (from env):", password)
	fmt.Println("Token (from env):", token)
}
