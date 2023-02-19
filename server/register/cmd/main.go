package main

import "goRTCServer/server/register/src"

func close() {
	src.Stop()
}
func main() {
	defer close()
	src.Start()
	select {}
}
