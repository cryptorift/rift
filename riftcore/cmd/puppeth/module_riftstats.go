// Copyright 2017 The CryptoRift Authors
// This file is part of riftcore.
//
// riftcore is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// riftcore is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with riftcore. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/cryptorift/riftcore/log"
)

// riftstatsDockerfile is the Dockerfile required to build an riftstats backend
// and associated monitoring site.
var riftstatsDockerfile = `
FROM mhart/alpine-node:latest

RUN \
  apk add --update git                                         && \
  git clone --depth=1 https://github.com/karalabe/rift-netstats && \
	apk del git && rm -rf /var/cache/apk/*                       && \
	\
  cd /rift-netstats && npm install && npm install -g grunt-cli && grunt

WORKDIR /rift-netstats
EXPOSE 3000

RUN echo 'module.exports = {trusted: [{{.Trusted}}], banned: []};' > lib/utils/config.js

CMD ["npm", "start"]
`

// riftstatsComposefile is the docker-compose.yml file required to deploy and
// maintain an riftstats monitoring site.
var riftstatsComposefile = `
version: '2'
services:
  riftstats:
    build: .
    image: {{.Network}}/riftstats{{if not .VHost}}
    ports:
      - "{{.Port}}:3000"{{end}}
    environment:
      - WS_SECRET={{.Secret}}{{if .VHost}}
      - VIRTUAL_HOST={{.VHost}}{{end}}
    logging:
      driver: "json-file"
      options:
        max-size: "1m"
        max-file: "10"
    restart: always
`

// deployRiftstats deploys a new riftstats container to a remote machine via SSH,
// docker and docker-compose. If an instance with the specified network name
// already exists there, it will be overwritten!
func deployRiftstats(client *sshClient, network string, port int, secret string, vhost string, trusted []string) ([]byte, error) {
	// Generate the content to upload to the server
	workdir := fmt.Sprintf("%d", rand.Int63())
	files := make(map[string][]byte)

	for i, address := range trusted {
		trusted[i] = fmt.Sprintf("\"%s\"", address)
	}

	dockerfile := new(bytes.Buffer)
	template.Must(template.New("").Parse(riftstatsDockerfile)).Execute(dockerfile, map[string]interface{}{
		"Trusted": strings.Join(trusted, ", "),
	})
	files[filepath.Join(workdir, "Dockerfile")] = dockerfile.Bytes()

	composefile := new(bytes.Buffer)
	template.Must(template.New("").Parse(riftstatsComposefile)).Execute(composefile, map[string]interface{}{
		"Network": network,
		"Port":    port,
		"Secret":  secret,
		"VHost":   vhost,
	})
	files[filepath.Join(workdir, "docker-compose.yaml")] = composefile.Bytes()

	// Upload the deployment files to the remote server (and clean up afterwards)
	if out, err := client.Upload(files); err != nil {
		return out, err
	}
	defer client.Run("rm -rf " + workdir)

	// Build and deploy the riftstats service
	return nil, client.Stream(fmt.Sprintf("cd %s && docker-compose -p %s up -d --build", workdir, network))
}

// riftstatsInfos is returned from an riftstats status check to allow reporting
// various configuration parameters.
type riftstatsInfos struct {
	host   string
	port   int
	secret string
	config string
}

// String implements the stringer interface.
func (info *riftstatsInfos) String() string {
	return fmt.Sprintf("host=%s, port=%d, secret=%s", info.host, info.port, info.secret)
}

// checkRiftstats does a health-check against an riftstats server to verify whether
// it's running, and if yes, gathering a collection of useful infos about it.
func checkRiftstats(client *sshClient, network string) (*riftstatsInfos, error) {
	// Inspect a possible riftstats container on the host
	infos, err := inspectContainer(client, fmt.Sprintf("%s_riftstats_1", network))
	if err != nil {
		return nil, err
	}
	if !infos.running {
		return nil, ErrServiceOffline
	}
	// Resolve the port from the host, or the reverse proxy
	port := infos.portmap["3000/tcp"]
	if port == 0 {
		if proxy, _ := checkNginx(client, network); proxy != nil {
			port = proxy.port
		}
	}
	if port == 0 {
		return nil, ErrNotExposed
	}
	// Resolve the host from the reverse-proxy and configure the connection string
	host := infos.envvars["VIRTUAL_HOST"]
	if host == "" {
		host = client.server
	}
	secret := infos.envvars["WS_SECRET"]
	config := fmt.Sprintf("%s@%s", secret, host)
	if port != 80 && port != 443 {
		config += fmt.Sprintf(":%d", port)
	}
	// Run a sanity check to see if the port is reachable
	if err = checkPort(host, port); err != nil {
		log.Warn("Riftstats service seems unreachable", "server", host, "port", port, "err", err)
	}
	// Container available, assemble and return the useful infos
	return &riftstatsInfos{
		host:   host,
		port:   port,
		secret: secret,
		config: config,
	}, nil
}
