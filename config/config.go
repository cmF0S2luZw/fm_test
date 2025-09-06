package config

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Target struct {
	Path    string `json:"path" yaml:"path"`
	Exclude string `json:"exclude,omitempty" yaml:"exclude,omitempty"`
}

type Packet struct {
	Name    string   `json:"name" yaml:"name"`
	Ver     string   `json:"ver" yaml:"ver"`
	Targets []Target `json:"targets" yaml:"targets"`
	Packets []Packet `json:"packets,omitempty" yaml:"packets,omitempty"`
}

type Packages struct {
	Packages []Packet `json:"packages" yaml:"packages"`
}

func LoadPacketConfig(path string) (*Packet, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var packet Packet
	ext := filepath.Ext(path)
	if ext == ".yaml" || ext == ".yml" {
		err = yaml.Unmarshal(data, &packet)
	} else {
		err = json.Unmarshal(data, &packet)
	}
	if err != nil {
		return nil, err
	}

	return &packet, nil
}

func LoadPackagesConfig(path string) (*Packages, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkgs Packages
	ext := filepath.Ext(path)
	if ext == ".yaml" || ext == ".yml" {
		err = yaml.Unmarshal(data, &pkgs)
	} else {
		err = json.Unmarshal(data, &pkgs)
	}
	if err != nil {
		return nil, err
	}

	return &pkgs, nil
}
