package main

import (
	"cache/cache"
	"fmt"
	"time"
)

func main() {
	defaultExpiration, _ := time.ParseDuration("0.5h")
	gcInterval, _ := time.ParseDuration("3s")
	c := cache.NewCache(defaultExpiration, gcInterval)

	k1 := "hello world"
	expiration, _ := time.ParseDuration("5s")

	c.Set("k1", k1, expiration)
	s, _ := time.ParseDuration("10s")
	if v, found := c.Get("k1"); found {
		fmt.Println("Found k1: ", v)
	} else {
		fmt.Println("Not found k1")
	}

	// sleep 10s
	time.Sleep(s)

	// 此时k1已经被清理掉了，测试一下：
	if v, found := c.Get("k1"); found {
		fmt.Println("Found k1: ", v)
	} else {
		fmt.Println("Not found k1")
	}
}
