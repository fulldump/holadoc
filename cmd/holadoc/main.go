package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/fulldump/goconfig"

	"github.com/fulldump/holadoc"
)

var VERSION = "dev"

func main() {
	c := holadoc.Config{
		Src:       "src/",
		Www:       "www/",
		Versions:  "v1,v2",
		Languages: "en,es,zh",
	}
	goconfig.Read(&c)

	if c.Version {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	if c.Serve != "" {

		s := &http.Server{
			Addr:    c.Serve,
			Handler: http.FileServer(http.Dir(c.Www)),
		}

		s.ListenAndServe()
		return
	}
}
