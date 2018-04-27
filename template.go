package main

import (
	"fmt"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"crypto/sha256"
	"strings"
	"os/exec"
	"os"
	"text/template"
	"path/filepath"
	"bytes"
	log "github.com/Sirupsen/logrus"
)

type rancherTemplate struct {
	Name 			string 	 `description:"Template name"`
	Hash 			string 	 `description:"Template data hash"`
	Destination  	string 	 `description:"Template destination file" yaml:"destination,omitempty"`
	Source 	   	 	string   `description:"Template source file" yaml:"source,omitempty"`
	Action		 	string	 `description:"Template action if change" yaml:"action,omitempty"`
}

func (r *rancherTemplate) getConfig(file string) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.WithFields(log.Fields{"file": file, "error": err}).Error("Failed reading config yaml.")
		return err
	}

	err = yaml.Unmarshal(content, &r)
	if err != nil {
		log.WithFields(log.Fields{"file": file, "error": err}).Error("Failed unmarshaling config yaml.")
	}

	_, r.Name = filepath.Split(r.Source)

	r.Hash = r.getDestinationHash()

	return err
}

func (r *rancherTemplate) getDestinationHash() string {
  	content, err := ioutil.ReadFile(r.Destination)
	if err != nil {
		return ""
	} 

  	return fmt.Sprintf("%x", sha256.Sum256(content))
}

func (r *rancherTemplate) getDataHash(w []byte) string {
  	return fmt.Sprintf("%x", sha256.Sum256(w))
}

func (r *rancherTemplate) getHash() string {
  	return r.Hash
}

func (r *rancherTemplate) updateHash(h string) bool {
	if r.hasChanged(h) {
		log.WithFields(log.Fields{"Old": r.Hash, "New": h}).Debug("Updating hash.")
		r.Hash = h
		return true
	}
	return false
}

func (r *rancherTemplate) hasChanged(h string) (bool) {
	if h != r.Hash {
  		return true
  	}

  	return false
}

func (r *rancherTemplate) doAction() {
	if r.Action != "" {
		log.WithField("action", r.Action).Info("Executing Action.")
		err := exec.Command("sh", "-c", r.Action).Run()
		if err != nil {
	        log.WithFields(log.Fields{"action": r.Action, "error": err}).Error("Failed executing action.")
	    }
	}
}

func (r *rancherTemplate) getTemplateFunc() template.FuncMap{
	return template.FuncMap{
		"split": func (s, sep string) []string {
			return strings.Split(s, sep)
		},
		"replace": func (s, old, new string) string {
			return strings.Replace(s, old, new , -1)
		},
		"tolower": func (s string) string {
			return strings.ToLower(s)
		},
		"getenv": func (s string) string {
			return os.Getenv(s)
		},
		"contains": func (s, c string) bool {
			return strings.Contains(s, c)
		},
		"ishealthy": func (s string) bool {
			return strings.Contains(s, "healthy")
		},
		"isrunning": func (s string) bool {
			return strings.Contains(s, "running")
		},
	}
}

func (r *rancherTemplate) execute(data interface{}) {
	log.WithField("file", r.Source).Debug("Executing template.")

	t, err := template.New(r.Name).Funcs(r.getTemplateFunc()).ParseFiles(r.Source)
	if err != nil {
		log.WithFields(log.Fields{"file": r.Source, "error": err}).Error("Failed parsing template.")
		return
	}

	var dest_buf bytes.Buffer
	err = t.Execute(&dest_buf, data)
	if err != nil {
		log.WithFields(log.Fields{"file": r.Source, "error": err}).Error("Failed executing template.")
		return
	}

	dest_bytes := dest_buf.Bytes()
	if r.updateHash(r.getDataHash(dest_bytes)) {
		err := ioutil.WriteFile(r.Destination, dest_bytes, 0644)
		if err != nil {
			log.WithFields(log.Fields{"file": r.Destination, "error": err}).Error("Failed writing file.")
			return
		}

		log.WithField("file", r.Destination).Info("Template has been updated")

		r.doAction()
	}
}

type rancherTemplates struct {
	rancherTemplates 	[]*rancherTemplate
}

func newRancherTemplates(files []string) *rancherTemplates{
	var temp = &rancherTemplates{}

	err := temp.getConfig(files)
	if err != nil {
		log.WithField("error", err).Error("Failed creating rancherTemplates.")
		return nil
	}

	return temp
}

func (r *rancherTemplates) execute(data interface{}) {
	for _ , tmpl := range r.rancherTemplates {
        tmpl.execute(data)
    }
}

func (r *rancherTemplates) getConfig(files []string) error {
	var err error
	for _, file := range files {
		var temp = &rancherTemplate{}

		err = temp.getConfig(file)
		if err == nil {
			r.rancherTemplates = append(r.rancherTemplates, temp)
		}
	}

	if len(files) ==0 { 
		log.Fatal("No template config file found.") 
	}

	return err
}
