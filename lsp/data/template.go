package main

import "fmt"
import "github.com/bittygarden/lilac/io_tool"
import "github.com/bits-and-blooms/bitset"

func main() {
	fmt.Println("Hello")
	fmt.Println(io_tool.FileExists("aa.txt"))
	fmt.Println(new(Person))
	fmt.Println(bitset.New(9))
}

type Person struct {
	name string
}
