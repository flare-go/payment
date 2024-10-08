package main

import "log"

func main() {

	server, err := InitializePaymentService()
	if err != nil {
		log.Fatal(err)
		return
	}

	if err = server.Run(":8080"); err != nil {
		log.Fatal(err.Error())
	}

}
