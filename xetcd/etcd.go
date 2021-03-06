package xetcd

import (
	"context"
	"fmt"
	"time"

	"github.com/sandwich-go/xconf/kv"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// Loader etcd Loader
type Loader struct {
	cli *clientv3.Client
	*kv.Common
}

// New new etcd Loader
func New(endpoint []string, opts ...kv.Option) (p kv.Loader, err error) {
	x := &Loader{}
	x.Common = kv.New("etcd", x, opts...)
	x.cli, err = getEtcdClient(endpoint)
	return x, err
}

// todo 连接option设定
func getEtcdClient(endpoint []string) (*clientv3.Client, error) {
	if len(endpoint) == 0 {
		return nil, fmt.Errorf("got empty endpoint")
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoint,
		DialTimeout: 5 * time.Second,
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	})
	if err != nil {
		return nil, fmt.Errorf("got error when new client, err:%w", err)
	}
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = cli.Status(timeoutCtx, endpoint[0])
	if err != nil {
		return nil, fmt.Errorf("got error when checking etcd status, err:%w", err)
	}
	return cli, nil
}

// CloseImplement 实现common.loaderImplement.CloseImplement
func (p *Loader) CloseImplement(ctx context.Context) error {
	return p.cli.Close()
}

// GetImplement 实现common.loaderImplement.GetImplement
func (p *Loader) GetImplement(ctx context.Context, confPath string) ([]byte, error) {
	ret, err := p.cli.Get(ctx, confPath)
	if err != nil {
		return nil, fmt.Errorf("got error:%v when Get with path:%s", confPath, err)
	}
	if len(ret.Kvs) == 0 {
		return nil, fmt.Errorf("got empty kv response with path:%s", confPath)
	}
	return ret.Kvs[0].Value, nil
}

// WatchImplement 实现common.loaderImplement.WatchImplement
func (p *Loader) WatchImplement(ctx context.Context, confPath string, onContentChange kv.ContentChange) {
	wc := clientv3.NewWatcher(p.cli)
	defer func() { _ = wc.Close() }()
	watchChan := wc.Watch(ctx, confPath, clientv3.WithPrefix())
	go func(pin *Loader, oc kv.ContentChange) {
		for resp := range watchChan {
			select {
			case <-pin.Done:
				return
			default:
			}
			for _, ev := range resp.Events {
				oc(pin.Name(), (string)(ev.Kv.Key), ev.Kv.Value)
			}
		}
	}(p, onContentChange)
}
