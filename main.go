package main

import (
	"flag"
	"log"
	"math"
	"time"

	"github.com/fagongzi/gateway/pkg/client"
	"github.com/fagongzi/gateway/pkg/pb/metapb"
)

var (
	addrAPI = flag.String("addr-api", "127.0.0.1:9092", "Addr: new gateway api server address")
	oldCfg  = flag.String("old", "", "Cfg: old gateway proxy config file")
)

var (
	oldClusters = make(map[string]*Cluster)
	newClusters = make(map[string]uint64)

	oldServers = make(map[string]*Server)
	newServers = make(map[string]uint64)
)

func main() {
	flag.Parse()

	cfg := getCfg(*oldCfg)
	store, err := getStoreFrom(cfg.RegistryAddr, cfg.Prefix)
	if err != nil {
		log.Fatalf("create old store failed, errors:\n%+v", err)
	}

	newCli, err := getClient(*addrAPI)
	if err != nil {
		log.Fatalf("create new api failed, errors:\n%+v", err)
	}

	migrateCluster(store, newCli)
	migrateServers(store, newCli)
	migrateAPI(store, newCli)

	log.Printf("migrate complete.")
}

func migrateCluster(store OldStore, newCli client.Client) {
	olds, err := store.GetClusters()
	if err != nil {
		log.Fatalf("get old clusters failed, errors:\n%+v", err)
	}

	log.Printf("migrate clusters started")
	log.Printf("load %d clusters", len(olds))
	for _, c := range olds {
		oldClusters[c.Name] = c
		id, err := newCli.NewClusterBuilder().Name(c.Name).Loadbalance(metapb.RoundRobin).Commit()
		if err != nil {
			log.Fatalf("migrate old cluster<%+v> failed, errors:\n%+v", c, err)
		}

		newClusters[c.Name] = id
	}
	log.Printf("migrate clusters completed")
}

func migrateServers(store OldStore, newCli client.Client) {
	olds, err := store.GetServers()
	if err != nil {
		log.Fatalf("get old servers failed, errors:\n%+v", err)
	}

	log.Printf("migrate servers started")
	log.Printf("load %d servers", len(olds))
	for _, s := range olds {
		oldServers[s.Addr] = s
		sb := newCli.NewServerBuilder().Addr(s.Addr)
		if s.MaxQPS == 0 {
			sb.MaxQPS(math.MaxInt64)
		} else {
			sb.MaxQPS(int64(s.MaxQPS))
		}

		if s.CheckPath != "" && s.CheckResponsedBody == "" {
			sb.CheckHTTPCode(s.CheckPath, time.Second*time.Duration(s.CheckDuration), time.Second*time.Duration(s.CheckTimeout))
		} else if s.CheckPath != "" && s.CheckResponsedBody != "" {
			sb.CheckHTTPBody(s.CheckPath, s.CheckResponsedBody, time.Second*time.Duration(s.CheckDuration), time.Second*time.Duration(s.CheckTimeout))
		}

		if s.External {
			sb.NoHeathCheck()
		}

		if s.HalfToOpenCollectSeconds > 0 &&
			s.OpenToCloseCollectSeconds > 0 &&
			s.HalfToOpenSeconds > 0 &&
			s.HalfToOpenSucceedRate > 0 &&
			s.HalfTrafficRate > 0 &&
			s.OpenToCloseFailureRate > 0 {
			sb.CircuitBreakerHalfTrafficRate(s.HalfTrafficRate)
			sb.CircuitBreakerCheckPeriod(time.Second * time.Duration(s.HalfToOpenCollectSeconds))
			sb.CircuitBreakerCloseToHalfTimeout(time.Second * time.Duration(s.HalfToOpenSeconds))
			sb.CircuitBreakerHalfToCloseCondition(s.OpenToCloseFailureRate)
			sb.CircuitBreakerHalfToOpenCondition(s.HalfToOpenSucceedRate)
		}

		id, err := sb.Commit()
		if err != nil {
			log.Fatalf("migrate old server<%+v> failed, errors:\n%+v", s, err)
		}

		newServers[s.Addr] = id
	}
	log.Printf("migrate servers completed")
}

func migrateBind(store OldStore, newCli client.Client) {
	olds, err := store.GetBinds()
	if err != nil {
		log.Fatalf("get old binds failed, errors:\n%+v", err)
	}

	log.Printf("migrate binds started")
	log.Printf("load %d binds", len(olds))
	for _, b := range olds {
		cid := newClusters[b.ClusterName]
		sid := newServers[b.ServerAddr]

		if sid == 0 || cid == 0 {
			log.Printf("[warn] migrate old bind<%+v> failed, missing cluster or server", b)
			continue
		}

		err = newCli.AddBind(cid, sid)
		if err != nil {
			log.Fatalf("migrate old bind<%+v> failed, errors:\n%+v", b, err)
		}
	}
	log.Printf("migrate binds completed")
}

func migrateAPI(store OldStore, newCli client.Client) {
	olds, err := store.GetAPIs()
	if err != nil {
		log.Fatalf("get old apis failed, errors:\n%+v", err)
	}

	log.Printf("migrate apis started")
	log.Printf("load %d apis", len(olds))

	for _, a := range olds {
		ab := newCli.NewAPIBuilder().Name(a.Name).MatchURLPattern(a.URL).MatchMethod(a.Method).MatchDomain(a.Domain)
		if a.Status == int(metapb.Up) {
			ab.UP()
		} else if a.Status == int(metapb.Down) {
			ab.Down()
		}

		if a.AccessControl != nil {
			ab.AddWhitelist(a.AccessControl.Whitelist...)
			ab.AddBlacklist(a.AccessControl.Blacklist...)
		}

		if a.Mock != nil {
			ab.DefaultValue([]byte(a.Mock.Value))
			for _, h := range a.Mock.Headers {
				ab.AddDefaultValueHeader(h.Name, h.Value)
			}

			ab.AddDefaultValueHeader("Content-Type", a.Mock.ContentType)
		}

		for _, n := range a.Nodes {
			cid := newClusters[n.ClusterName]
			if cid == 0 {
				log.Printf("[warn] migrate old api<%+v> failed, missing cluster %s", a, n.ClusterName)
				continue
			}

			ab.AddDispatchNode(cid)
			if n.AttrName != "" {
				ab.DispatchNodeValueAttrName(cid, n.AttrName)
			}

			if n.Rewrite != "" {
				ab.DispatchNodeURLRewrite(cid, n.Rewrite)
			}

			for _, v := range n.Validations {
				source := metapb.QueryString

				if v.GetFrom == 1 {
					source = metapb.FormData
				}

				param := metapb.Parameter{
					Name:   v.Attr,
					Source: source,
				}

				for _, r := range v.Rules {
					ab.AddDispatchNodeValidation(cid, param, r.Expression, v.Required)
				}
			}
		}

		_, err = ab.Commit()
		if err != nil {
			log.Fatalf("migrate old API<%+v> failed, errors:\n%+v", a, err)
		}
	}

	log.Printf("migrate apis completed")
}

func getClient(addr string) (client.Client, error) {
	return client.NewClient(time.Second*10, addr)
}
