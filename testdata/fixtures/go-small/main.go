package main

import "fmt"

type Greeter struct {
	name string
}

func (g *Greeter) Hello() string {
	return fmt.Sprintf("hi %s", g.name)
}

func main() {
	g := &Greeter{name: "world"}
	fmt.Println(g.Hello())
}

func unused() {}
