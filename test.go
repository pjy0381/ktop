package main

import (
    "fmt"
    "io"
    "os"
)

func main() {
    // /etc/hosts 파일 열기
    file, err := os.Open("/etc/hosts")
    if err != nil {
        // 파일 열기 실패
        fmt.Println("Error opening file:", err)
        return
    }
    defer file.Close()

    // 파일 내용 읽기
    content, err := io.ReadAll(file)
    if err != nil {
        // 파일 읽기 실패
        fmt.Println("Error reading file:", err)
        return
    }

    // 파일 내용 출력
    fmt.Println(string(content))
}

