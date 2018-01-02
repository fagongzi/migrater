package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/fagongzi/util/json"
	"github.com/hashicorp/consul/api"
)

var (
	supportSchema = make(map[string]func(string, string) (OldStore, error))
)

func init() {
	supportSchema["consul"] = getConsulStoreFrom
	supportSchema["etcd"] = getEtcdStoreFrom
}

func getStoreFrom(registryAddr, prefix string) (OldStore, error) {
	u, err := url.Parse(registryAddr)
	if err != nil {
		panic(fmt.Sprintf("parse registry addr failed, errors:%+v", err))
	}

	schema := strings.ToLower(u.Scheme)
	fn, ok := supportSchema[schema]
	if ok {
		return fn(u.Host, prefix)
	}

	return nil, fmt.Errorf("not support: %s", registryAddr)
}

func getConsulStoreFrom(addr, prefix string) (OldStore, error) {
	return newConsulStore(addr, prefix)
}

func getEtcdStoreFrom(addr, prefix string) (OldStore, error) {
	var addrs []string
	values := strings.Split(addr, ",")

	for _, value := range values {
		addrs = append(addrs, fmt.Sprintf("http://%s", value))
	}

	return newEtcdStore(addrs, prefix)
}

type consulStore struct {
	consulAddr string
	client     *api.Client

	prefix      string
	clustersDir string
	serversDir  string
	bindsDir    string
	apisDir     string
	proxiesDir  string
	routingsDir string
}

func newConsulStore(consulAddr string, prefix string) (OldStore, error) {
	if strings.HasPrefix(prefix, "/") {
		prefix = prefix[1:]
	}

	store := &consulStore{
		consulAddr:  consulAddr,
		prefix:      prefix,
		clustersDir: fmt.Sprintf("%s/clusters", prefix),
		serversDir:  fmt.Sprintf("%s/servers", prefix),
		bindsDir:    fmt.Sprintf("%s/binds", prefix),
		apisDir:     fmt.Sprintf("%s/apis", prefix),
		proxiesDir:  fmt.Sprintf("%s/proxy", prefix),
		routingsDir: fmt.Sprintf("%s/routings", prefix),
	}

	conf := api.DefaultConfig()
	conf.Address = consulAddr

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, err
	}

	store.client = client
	if err != nil {
		return nil, err
	}

	return store, nil
}

func (s *consulStore) GetBinds() ([]*Bind, error) {
	pairs, _, err := s.client.KV().List(s.bindsDir, nil)

	if nil != err {
		return nil, err
	}

	values := make([]*Bind, len(pairs))
	i := 0

	for _, pair := range pairs {
		key := strings.Replace(pair.Key, fmt.Sprintf("%s/", s.bindsDir), "", 1)
		infos := strings.SplitN(key, "-", 2)

		values[i] = &Bind{
			ServerAddr:  infos[0],
			ClusterName: infos[1],
		}

		i++
	}

	return values, nil
}

func (s *consulStore) GetClusters() ([]*Cluster, error) {
	pairs, _, err := s.client.KV().List(s.clustersDir, nil)

	if nil != err {
		return nil, err
	}

	values := make([]*Cluster, len(pairs))
	i := 0

	for _, pair := range pairs {
		value := &Cluster{}
		json.MustUnmarshal(value, pair.Value)
		values[i] = value

		i++
	}

	return values, nil
}

func (s *consulStore) GetAPIs() ([]*API, error) {
	pairs, _, err := s.client.KV().List(s.apisDir, nil)

	if nil != err {
		return nil, err
	}

	values := make([]*API, len(pairs))
	i := 0

	for _, pair := range pairs {
		value := &API{}
		json.MustUnmarshal(value, pair.Value)
		values[i] = value
		i++
	}

	return values, nil
}

func (s *consulStore) GetServers() ([]*Server, error) {
	pairs, _, err := s.client.KV().List(s.serversDir, nil)

	if nil != err {
		return nil, err
	}

	values := make([]*Server, len(pairs))
	i := 0

	for _, pair := range pairs {
		value := &Server{}
		json.MustUnmarshal(value, pair.Value)
		values[i] = value
		i++
	}

	return values, nil
}

type etcdStore struct {
	cli         *clientv3.Client
	prefix      string
	clustersDir string
	serversDir  string
	bindsDir    string
	apisDir     string
	proxiesDir  string
	routingsDir string
}

func newEtcdStore(etcdAddrs []string, prefix string) (OldStore, error) {
	store := &etcdStore{
		prefix:      prefix,
		clustersDir: fmt.Sprintf("%s/clusters", prefix),
		serversDir:  fmt.Sprintf("%s/servers", prefix),
		bindsDir:    fmt.Sprintf("%s/binds", prefix),
		apisDir:     fmt.Sprintf("%s/apis", prefix),
		proxiesDir:  fmt.Sprintf("%s/proxy", prefix),
		routingsDir: fmt.Sprintf("%s/routings", prefix),
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdAddrs,
		DialTimeout: time.Second * 10,
	})

	if err != nil {
		return nil, err
	}

	store.cli = cli
	return store, nil
}

func (e *etcdStore) GetAPIs() ([]*API, error) {
	var values []*API
	err := e.getList(e.apisDir, func(item *mvccpb.KeyValue) {
		value := &API{}
		json.MustUnmarshal(value, item.Value)
		values = append(values, value)
	})

	return values, err
}

func (e *etcdStore) GetServers() ([]*Server, error) {
	var values []*Server
	err := e.getList(e.serversDir, func(item *mvccpb.KeyValue) {
		value := &Server{}
		json.MustUnmarshal(value, item.Value)
		values = append(values, value)
	})

	return values, err
}

func (e *etcdStore) GetClusters() ([]*Cluster, error) {
	var values []*Cluster
	err := e.getList(e.clustersDir, func(item *mvccpb.KeyValue) {
		value := &Cluster{}
		json.MustUnmarshal(value, item.Value)
		values = append(values, value)
	})

	return values, err
}

func (e *etcdStore) GetBinds() ([]*Bind, error) {
	var values []*Bind
	err := e.getList(e.bindsDir, func(item *mvccpb.KeyValue) {
		value := &Bind{}
		json.MustUnmarshal(value, item.Value)
		values = append(values, value)
	})

	return values, err
}

func (e *etcdStore) getList(key string, fn func(*mvccpb.KeyValue)) error {
	ctx, cancel := context.WithTimeout(e.cli.Ctx(), time.Second*30)
	defer cancel()

	resp, err := clientv3.NewKV(e.cli).Get(ctx, key, clientv3.WithPrefix())
	if nil != err {
		return err
	}

	for _, item := range resp.Kvs {
		fn(item)
	}

	return nil
}
