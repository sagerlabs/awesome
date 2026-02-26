package main

import "fmt"

// StringUtils 字符串工具函数
func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Add 简单的加法函数
func Add(a, b int) int {
	return a + b
}

func main() {
	fmt.Println("Hello, World!")
	fmt.Println("Reverse of 'hello':", ReverseString("hello"))
	fmt.Println("1 + 2 =", Add(1, 2))
}
