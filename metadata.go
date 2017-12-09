package main

import (
	"fmt"
	"os"
	"io"
	"os/signal"
	"sync"
	"sort"
	"syscall"
	log "github.com/Sirupsen/logrus"
	rancher "github.com/rancher/go-rancher-metadata/metadata"
)

type rancherMetadataConfig struct {
	Url 		string 	`description:"Url used for accessing the Rancher metadata service"`
	Prefix      string 	`description:"Prefix used for accessing the Rancher metadata service"`
	Refresh 	int   	`description:"Refresh the Rancher metadata service every 'Refresh' seconds"`
	Self		bool	`description:"Get self data or all"`
}

type rancherMetadataData []rancher.Stack
type rancherMetadataService []rancher.Service

type rancherMetadata struct {		
	Cli 		rancher.Client 			`description:"Rancher metadata client"`
	Config 		rancherMetadataConfig 	`description:"Rancher metadata configuration"`
	Templates	*rancherTemplates		`description:"Templates configuration"`
	Input 		chan *rancherMetadataData
	Exit 		chan os.Signal
	Runners		[]chan struct{}
	Logfile 	string
}


func newMetadata(conf Params) *rancherMetadata {
	var m = &rancherMetadata{
		Runners: []chan struct{}{},
	}

	m.Input = make(chan *rancherMetadataData,1)
	m.Exit = make(chan os.Signal, 1)
	signal.Notify(m.Exit, syscall.SIGINT, syscall.SIGABRT, syscall.SIGKILL, syscall.SIGTERM)

	m.Templates = newRancherTemplates(conf.Files)

	m.Config.Url = conf.Url
	m.Config.Prefix = conf.Prefix
	m.Config.Refresh = conf.Refresh
	m.Config.Self = conf.Self
	m.Logfile = conf.Logfile

	customFormatter := new(log.TextFormatter)
    customFormatter.TimestampFormat = "2006-01-02 15:04:05"
    log.SetFormatter(customFormatter)
    if conf.Debug {
    	log.SetLevel(log.DebugLevel)
    }
    customFormatter.FullTimestamp = true

	return m
}

func (r rancherMetadataService) Len() int { 
	return len(r) 
}

func (r rancherMetadataService) Swap(i, j int) { 
	r[i], r[j] = r[j], r[i]
}

func (r rancherMetadataService) Less(i, j int) bool { 
	return r[i].Name < r[j].Name 
}

func (r rancherMetadataData) Len() int { 
	return len(r) 
}

func (r rancherMetadataData) Swap(i, j int) { 
	r[i], r[j] = r[j], r[i]
}

func (r rancherMetadataData) Less(i, j int) bool { 
	return r[i].Name < r[j].Name 
}

func (r rancherMetadataData) Sort() { 
	sort.Sort(r) 

	for _, data := range r {
		sort.Sort(rancherMetadataService(data.Services))
	}
}

func (m *rancherMetadata) connect() error {
	metadataServiceURL := fmt.Sprintf("%s/%s", m.Config.Url, m.Config.Prefix)

	log.WithField("url", metadataServiceURL).Info("Connecting to Rancher metadata.")

	client, err := rancher.NewClientAndWait(metadataServiceURL)

	if err != nil {
		log.WithFields(log.Fields{"url": metadataServiceURL,"error": err}).Errorln("Failed connecting to Rancher metadata.")
	} else {
		m.Cli = client
	}

	return err
}

func (m *rancherMetadata) update() func(string) {
	update := func(version string) {
		var err error
		var stacks rancherMetadataData

		log.WithField("version", version).Debug("Refreshing configuration from Rancher metadata service")

		if m.Config.Self {
			var stack rancher.Stack

			stack, err = m.Cli.GetSelfStack()
			stacks = append(stacks, stack)
		} else {
			stacks, err = m.Cli.GetStacks()
		}

		if err != nil {
			log.WithField("error", err).Error("Failed getting Rancher metadata data.", err)
			return
		}

		stacks.Sort()

		m.Input <- &stacks
	}

	update("init")
	return update
}

func (m *rancherMetadata) onChange(stop chan struct{}) {	
	log.Info("Listening for Rancher metadata data")
	go m.Cli.OnChange(m.Config.Refresh, m.update())

	for {
        select {
        case <- stop:
            return
        }
    }
}

func (m *rancherMetadata) writeTemplates() {
	for {
        select {
        case stacks := <- m.Input:
        	if stacks != nil {
            	m.Templates.execute(*stacks)
        	} else {
        		return
        	}
        }
    }
}

func (m *rancherMetadata) addRunner() chan struct{} {
	chan_new := make(chan struct{}, 1)
	m.Runners = append(m.Runners, chan_new)

	return chan_new
}

func (m *rancherMetadata) closeRunners() {
	for _, r_chan := range m.Runners {
		if r_chan != nil {
			r_chan <- struct{}{}
		}
	}
	m.Runners = nil
}

func (m *rancherMetadata) run() {
	var in, out sync.WaitGroup
	indone := make(chan struct{},1)
	outdone := make(chan struct{},1)

	log.Info("Running Rancher metadata")

	in.Add(1)
	go func() {
		defer in.Done()
		m.onChange(m.addRunner())
	}()

	go func() {
		in.Wait()
		close(m.Input)
		close(indone)
	}()

	out.Add(1)
	go func() {
		defer out.Done()
		m.writeTemplates()
	}()

	go func() {
		out.Wait()
		close(outdone)
	}()

	for {
        select {
        case <- indone:
        	<- outdone
        	return
        case <- outdone:
        	log.Error("Aborting...")
        	m.closeRunners()
        	return
        case <- m.Exit:
        	log.Warn("Exit signal detected....Closing...")
        	close(m.Exit)
        	go m.closeRunners()
        	select {
        	case <- outdone:
        		return
        	}
        }
    }
}

func (m *rancherMetadata) init() {
	log_file, err := os.OpenFile(m.Logfile, os.O_WRONLY | os.O_CREATE, 0644)
	if err != nil {
    	log.WithFields(log.Fields{"file": m.Logfile, "error": err}).Fatal("Failed opening log file.")
	}
	defer log_file.Close()

	multi := io.MultiWriter(log_file, os.Stdout)

	log.SetOutput(multi)

	log.Info("Initializing Rancher metadata")

	err = m.connect()
	if err != nil {
		return
	}

	m.run()

}


