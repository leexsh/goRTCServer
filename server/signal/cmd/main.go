package main

import "goRTCServer/server/signal/src"

func close() {
	src.Stop()
}
func main() {
	defer close()
	src.Start()
	select {}
}
