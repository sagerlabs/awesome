package main

import (
	"fmt"
	"os"
)

// 这段代码有多个问题
func main() {
	// 问题1: 硬编码密码
	password := "my-secret-password-123"
	token := "ghp_this_is_a_secret_token"
	
	// 问题2: 未处理错误
	file, _ := os.Open("config.txt")
	defer file.Close()
	
	// 问题3: nil 指针风险
	var data *string
	fmt.Println(*data) // 这里会 panic
	
	fmt.Println("Password:", password)
	fmt.Println("Token:", token)
}
