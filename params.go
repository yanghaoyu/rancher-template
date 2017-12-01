package main

import (
	"flag"
	"path/filepath"
	log "github.com/Sirupsen/logrus"
)

func check(e error, m string) {
    if e != nil {
		log.Error("[Error]: ", m , e)
	}
}

type Params struct {
	Url string
	Prefix string
	Logfile string
	Files []string
	Refresh int
	Self bool
	Debug bool
}

func (p *Params) init() {
	var file_path string

	flag.StringVar(&p.Url, "url", "http://rancher-metadata.rancher.internal", "Rancher metadata url.")
	flag.StringVar(&p.Prefix, "prefix", "2016-07-29", "Rancher metadata prefix.")
	flag.StringVar(&p.Logfile, "logfile", "/proc/1/fd/1", "Rancher template log fie.")
	flag.StringVar(&file_path, "templates", "/opt/tools/rancher-template/etc/*.yml", "Template files, wildcard allowed between quotes.")
	flag.IntVar(&p.Refresh, "refresh", 300, "Rancher metadata refresh time in seconds.")
	flag.BoolVar(&p.Self, "self", false, "Get self stack data or all.")
	flag.BoolVar(&p.Debug, "debug", false, "Run in debug mode.")

	flag.Parse()

	p.getFiles(file_path)
}

func (p *Params) getFiles(f string) {
	var err error

	p.Files , err = filepath.Glob(f)
	if err != nil {
		log.Fatal(err)
	}
}