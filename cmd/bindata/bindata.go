package main

import (
	"github.com/go-bindata/go-bindata"
)

func main() {
	input := bindata.InputConfig{"data", true}
	fn := "bound/bound.go"
	c := bindata.NewConfig()
	c.Package = "bound"
	c.Output = fn
	c.Input = []bindata.InputConfig{input}
	c.Prefix = "data"
	c.NoMetadata = true
	c.NoCompress = true
	if err := bindata.Translate(c); err != nil {
		panic(err)
	}
}
