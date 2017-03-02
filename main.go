package main

func main() {
	server := NewServer(":3000")
	StartServer(server)
}
