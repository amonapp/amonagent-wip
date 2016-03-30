package main

import "time"

func IsReady(what string, seconds int64, ch chan<- bool) {
	time.Sleep(time.Duration(1) * time.Second) // nanoseconds
	println(what, "is ready!")
	ch <- true
}

func main() {
	ch := make(chan bool)
	println("Let's go!")
	go IsReady("Coffee", 6, ch)
	go IsReady("Tea", 2, ch)
	println("I'm done here.")

	for i := 0; i < 2; i++ {
		<-ch
	}
}
