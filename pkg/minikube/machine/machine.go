/*
Copyright 2019 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package machine

import (
	"fmt"
	"time"

	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/host"
	libprovision "github.com/docker/machine/libmachine/provision"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/minikube/pkg/minikube/config"
	"k8s.io/minikube/pkg/minikube/driver"
	"k8s.io/minikube/pkg/provision"
)

// Machine contains information about a machine
type Machine struct {
	*host.Host
}

// IsValid checks if the machine has the essential info needed for a machine
func (h *Machine) IsValid() bool {
	if h == nil {
		return false
	}

	if h.Host == nil {
		return false
	}

	if h.Host.Name == "" {
		return false
	}

	if h.Host.Driver == nil {
		return false
	}

	if h.Host.HostOptions == nil {
		return false
	}

	if h.Host.RawDriver == nil {
		return false
	}
	return true
}

// LoadMachine returns a Machine abstracting a libmachine.Host
func LoadMachine(name string) (*Machine, error) {
	api, err := NewAPIClient()
	if err != nil {
		return nil, err
	}

	h, err := LoadHost(api, name)
	if err != nil {
		return nil, err
	}

	var mm Machine
	if h != nil {
		mm.Host = h
	} else {
		return nil, errors.New("host is nil")
	}
	return &mm, nil
}

// provisionDockerMachine provides fast provisioning of a docker machine
func provisionDockerMachine(h *host.Host) error {
	glog.Infof("provisioning docker machine ...")
	start := time.Now()
	defer func() {
		glog.Infof("provisioned docker machine in %s", time.Since(start))
	}()

	p, err := fastDetectProvisioner(h)
	if err != nil {
		return errors.Wrap(err, "fast detect")
	}
	return p.Provision(*h.HostOptions.SwarmOptions, *h.HostOptions.AuthOptions, *h.HostOptions.EngineOptions)
}

// fastDetectProvisioner provides a shortcut for provisioner detection
func fastDetectProvisioner(h *host.Host) (libprovision.Provisioner, error) {
	d := h.Driver.DriverName()
	switch {
	case driver.IsKIC(d):
		return provision.NewUbuntuProvisioner(h.Driver), nil
	case driver.BareMetal(d):
		return libprovision.DetectProvisioner(h.Driver)
	default:
		return provision.NewBuildrootProvisioner(h.Driver), nil
	}
}

func saveHost(api libmachine.API, h *host.Host, cfg *config.ClusterConfig, n *config.Node) error {
	if err := api.Save(h); err != nil {
		return errors.Wrap(err, "save")
	}

	// Save IP to config file for subsequent use
	ip, err := h.Driver.GetIP()
	if err != nil {
		return err
	}
	n.IP = ip
	fmt.Printf("SAVING NEW IP HERE: %s\n", ip)
	return config.SaveNode(cfg, n)
}
