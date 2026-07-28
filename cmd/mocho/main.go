// Command mocho serves the LLM-wiki study app: a React SPA and JSON API in a
// single binary, backed by a markdown wiki directory.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/rigofekete/mocho/internal/config"
	"github.com/rigofekete/mocho/internal/server"
	"github.com/rigofekete/mocho/internal/wiki"
)

func main() {
	wikiFlag := flag.String("wiki", "", "path to the wiki directory (overrides $MOCHO_WIKI, config file, default)")
	addrFlag := flag.String("addr", "", "listen address (overrides $MOCHO_ADDR, config file, default)")
	scaffoldFlag := flag.Bool("scaffold", true, "scaffold the wiki at the configured path if it is empty")
	flag.Parse()

	cfg, err := config.Resolve(*wikiFlag, *addrFlag)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	w := wiki.Wiki{Root: cfg.WikiPath}
	if *scaffoldFlag && !w.Exists() {
		fmt.Printf("scaffolding new wiki at %s\n", cfg.WikiPath)
		if err := w.Scaffold(); err != nil {
			log.Fatalf("scaffold: %v", err)
		}
	}

	app := server.New(w)
	fmt.Printf("mocho serving http://%s (wiki: %s)\n", cfg.Addr, cfg.WikiPath)
	if err := http.ListenAndServe(cfg.Addr, app.Handler()); err != nil {
		log.Fatalf("serve: %v", err)
	}
}