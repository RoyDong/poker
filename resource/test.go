package main

import "log"

func main() {


    c := make(chan struct{}, 1)


    c <- struct{}{}

    log.Println(1)


    close(c)

    c <- struct{}{}


}

