package main

import (
	"log"

	"github.com/kislerdm/diagramastext/core"
)

func main() {
	const code = `@startuml
    a -> b
@enduml`

	client := core.NewPlantUMLClient()

	svg, err := client.GenerateSVG(code)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(string(svg))
}
