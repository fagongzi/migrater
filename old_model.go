package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

// Cfg old cfg
type Cfg struct {
	RegistryAddr string `json:"registryAddr"`
	Prefix       string `json:"prefix"`
}

func getCfg(file string) *Cfg {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("bootstrap: read config file <%s> failed, errors:\n%+v",
			file,
			err)
	}

	cnf := &Cfg{}
	err = json.Unmarshal(data, cnf)
	if err != nil {
		log.Fatalf("bootstrap: parse config file <%s> failed, errors:\n%+v",
			file,
			err)
	}

	return cnf
}

// Cluster old cluster def
type Cluster struct {
	Name     string `json:"name,omitempty"`
	LbName   string `json:"lbName,omitempty"`
	External bool   `json:"external,omitempty"`
}

// ValidationRule validation rule
type ValidationRule struct {
	Type       int    `json:"type, omitempty"`
	Expression string `json:"expression, omitempty"`
}

// Validation validate rule
type Validation struct {
	Attr     string            `json:"attr, omitempty"`
	GetFrom  int               `json:"getFrom, omitempty"`
	Required bool              `json:"required, omitempty"`
	Rules    []*ValidationRule `json:"rules, omitempty"`
}

// Node api dispatch node
type Node struct {
	ClusterName string        `json:"clusterName, omitempty"`
	Rewrite     string        `json:"rewrite, omitempty"`
	AttrName    string        `json:"attrName, omitempty"`
	Validations []*Validation `json:"validations, omitempty"`
}

// AccessControl access control
type AccessControl struct {
	Whitelist []string `json:"whitelist, omitempty"`
	Blacklist []string `json:"blacklist, omitempty"`
}

// Mock mock
type Mock struct {
	Value       string        `json:"value"`
	ContentType string        `json:"contentType, omitempty"`
	Headers     []*MockHeader `json:"headers, omitempty"`
	Cookies     []string      `json:"cookies, omitempty"`
}

// MockHeader header
type MockHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// API a api define
type API struct {
	Name          string         `json:"name, omitempty"`
	URL           string         `json:"url"`
	Method        string         `json:"method"`
	Domain        string         `json:"domain, omitempty"`
	Status        int            `json:"status, omitempty"`
	AccessControl *AccessControl `json:"accessControl, omitempty"`
	Mock          *Mock          `json:"mock, omitempty"`
	Nodes         []*Node        `json:"nodes"`
	Desc          string         `json:"desc, omitempty"`
}

// Bind a bind server and cluster
type Bind struct {
	ClusterName string `json:"clusterName,omitempty"`
	ServerAddr  string `json:"serverAddr,omitempty"`
}

// Server server
type Server struct {
	Schema string `json:"schema,omitempty"`
	Addr   string `json:"addr,omitempty"`

	// External external, e.g. create from external service discovery
	External bool `json:"external,omitempty"`
	// CheckPath begin with / checkpath, expect return CheckResponsedBody.
	CheckPath string `json:"checkPath,omitempty"`
	// CheckResponsedBody check url responsed http body, if not set, not check body
	CheckResponsedBody string `json:"checkResponsedBody"`
	// CheckDuration check interval, unit second
	CheckDuration int `json:"checkDuration,omitempty"`
	// CheckTimeout timeout to check server
	CheckTimeout int `json:"checkTimeout,omitempty"`
	// Status Server status
	Status int `json:"status,omitempty"`

	// MaxQPS the backend server max qps support
	MaxQPS                    int `json:"maxQPS,omitempty"`
	HalfToOpenSeconds         int `json:"halfToOpenSeconds,omitempty"`
	HalfTrafficRate           int `json:"halfTrafficRate,omitempty"`
	HalfToOpenSucceedRate     int `json:"halfToOpenSucceedRate,omitempty"`
	HalfToOpenCollectSeconds  int `json:"halfToOpenCollectSeconds,omitempty"`
	OpenToCloseFailureRate    int `json:"openToCloseFailureRate,omitempty"`
	OpenToCloseCollectSeconds int `json:"openToCloseCollectSeconds,omitempty"`
}

// OldStore old store
type OldStore interface {
	GetBinds() ([]*Bind, error)
	GetClusters() ([]*Cluster, error)
	GetServers() ([]*Server, error)
	GetAPIs() ([]*API, error)
}
