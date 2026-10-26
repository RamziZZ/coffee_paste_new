package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"coffee_paste_backend/pkg/password"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Masukkan password admin baru: ")
	newPassword, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Gagal membaca password:", err)
		return
	}

	newPassword = strings.TrimRight(newPassword, "\r\n")
	if len(newPassword) < 8 {
		fmt.Println("Password minimal 8 karakter.")
		return
	}

	hash, err := password.Hash(newPassword)
	if err != nil {
		fmt.Println("Gagal membuat hash:", err)
		return
	}

	fmt.Println("\nHash password baru:")
	fmt.Println(hash)
}
