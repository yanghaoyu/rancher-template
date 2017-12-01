[![](https://images.microbadger.com/badges/image/rawmind/rancher-template.svg)](https://microbadger.com/images/rawmind/rancher-template "Get your own image badge on microbadger.com")

rancher-template
=====================

rancher-template is a utility to get stacks info from rancher metadata service and execute golang templates to generate files. 

It's included at docker [rawmind/rancher-tools][rancher-tools] at `/opt/tools/rancher-template`

## Build statically

```
git clone -v <version> https://github.com/rawmind0/rancher-metadata.git
cd rancher-metadata
go get
CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o rancher-template
```

## Versions

- [0.1](https://github.com/rawmind0/rancher-template/blob/0.1/Dockerfile)


## Usage

Rancher-template get stacks info from rancher metadata service and execute golang templates to generate files. 

It refresh data in real time, updating templates if content change. An action could also be defined, `command` to be exectued when template content changes. 
E.g. Restart a service if config changes.

These are options and parameters that could be set.
```
Usage of rancher-template:
  -debug
      Run in debug mode.
  -logfile string
      Rancher template log fie. (default "/proc/1/fd/1")
  -prefix string
      Rancher metadata prefix. (default "2016-07-29")
  -refresh int
      Rancher metadata refresh time in seconds. (default 300)
  -self
      Get self stack data or all.
  -templates string
      Template files, wildcard allowed between quotes. (default "/opt/tools/rancher-template/etc/*.yml")
  -url string
      Rancher metadata url. (default "http://rancher-metadata.rancher.internal")
```

## Templates 

Templates should be configured with an yaml file, that rancher-template reads.
This config yml file format..

```
destination: <destination file>
source: <template file>
action: <command> 
```

- `destination` set the file to write template execution
- `source` set golang template file
- `action` command to execute if template has changed. Optional

Templates get rancher.stacks data from Rancher metadata. You could get all stacks or just selfstack. 

Template example to list Stack name, service name and service label if has specific values.

```
{{- range $stack := . }}
Stack name {{.Name}}
  {{- range $service := .Services }}
  Service name {{.Name}}
    {{- $traefik_label := (index .Labels "traefik.enable") -}}
    {{- if or (eq $traefik_label "true") (eq $traefik_label "stack") }}
    labels {{$traefik_label}}
    {{- end -}}
  {{- end -}}
{{- end }}
```

[rancher-tools]: https://github.com/rawmind0/rancher-tools
