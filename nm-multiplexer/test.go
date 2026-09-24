package main

import "fmt"

func main() {
	ch := make(chan int, 1)
	go func() {
		ch <- 1
	}()
	go func() {
		ch <- 1
	}()
	x := <-ch
	fmt.Printf("%v\n", x)
	fmt.Printf("%v\n", ch)

	for a := range ch {
		fmt.Printf("%v\n", a)
	}
}
