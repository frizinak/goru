package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/frizinak/goru/common"
)

func exit(err error) {
	if err == nil {
		return
	}

	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	var maxResults uint
	var all bool
	var noStress bool
	flag.UintVar(&maxResults, "n", 3, "max amount of results")
	flag.BoolVar(&all, "a", false, "include words without translation")
	flag.BoolVar(&noStress, "ns", false, "don't print stress mark")
	flag.Parse()

	query := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if query == "" {
		exit(errors.New("please provide a query"))
	}

	d, err := common.GetDict()
	exit(err)

	custom := `{{- define "gender" -}}{{ . }}{{- end -}}`
	if noStress {
		custom += `{{- define "wordStr" -}}
{{ clrGreen }} {{- unstressed . -}} {{ clrPop }}
{{- end -}}`
	}

	masterTpl, err := common.GetTpl()
	exit(err)

	tpl, err := masterTpl.Parse(custom)
	exit(err)

	results, _ := d.Search(query, all, int(maxResults))
	if len(results) == 0 {
		results, _ = d.SearchFuzzy(query, all, int(maxResults))
	}
	if len(results) == 0 {
		exit(errors.New("no results"))
	}
	exit(tpl.Execute(os.Stdout, results))
}
