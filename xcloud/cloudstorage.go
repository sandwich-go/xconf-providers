package xcloud

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sandwich-go/xconf/kv"
)

const (
	LoaderName = "cloud"
)

func New(opts ...Option) (p kv.Loader, err error) {
	opt := NewOptions(opts...)
	genEndPoint := type2GenEndpointFunc[opt.StorageType]
	endpoint, err := genEndPoint(opt)
	if err != nil {
		return nil, err
	}
	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(opt.AccessKey, opt.Secret, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	x := &Loader{
		cc:           opt,
		cli:          cli,
		onChanged:    make(map[string][]kv.ContentChange),
		lastModified: make(map[string]time.Time),
	}
	x.Common = kv.New(LoaderName, x, opt.KVOption...)
	go x.watchEvent()
	return x, err
}

// Loader etcd Loader
type Loader struct {
	cli *minio.Client
	*kv.Common
	mutex        sync.Mutex
	onChanged    map[string][]kv.ContentChange
	lastModified map[string]time.Time
	cc           *Options
}

func (l *Loader) CloseImplement(ctx context.Context) error {
	return nil
}

func (l *Loader) GetImplement(ctx context.Context, confPath string) ([]byte, error) {
	object, err := l.cli.GetObject(ctx, l.cc.Bucket, confPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer func(object *minio.Object) {
		_ = object.Close()
	}(object)
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, object)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (l *Loader) WatchImplement(ctx context.Context, confPath string, onContentChange kv.ContentChange) {
	l.mutex.Lock()
	if len(l.onChanged[confPath]) == 0 {
		if ll, err := l.getLastModified(ctx, confPath); err == nil {
			l.lastModified[confPath] = ll
		}
	}
	l.onChanged[confPath] = append(l.onChanged[confPath], onContentChange)
	l.mutex.Unlock()
}

func (l *Loader) watchEvent() {
	for true {
		select {
		case <-l.Done:
			return
		case <-time.After(time.Second):

		}
		l.mutex.Lock()
		for k, _ := range l.onChanged {
			select {
			case <-l.Done:
				return
			default:
			}
			fileLast, ok := l.lastModified[k]
			if !ok {
				if l.fileChange(k) {
					l.lastModified[k] = time.Now()
				}
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			if ll, err := l.getLastModified(ctx, k); err == nil {
				if ll.After(fileLast) {
					if l.fileChange(k) {
						l.lastModified[k] = ll
					}
				}
			}
			cancel()
		}
		l.mutex.Unlock()
	}
}

func (l *Loader) fileChange(name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if b, err := l.Get(ctx, name); err == nil {
		if l.IsChanged(name, b) {
			for _, callback := range l.onChanged[name] {
				if errLoad := callback(LoaderName, name, b); errLoad == nil {
					l.cc.OnUpdate(name, b)
				} else {
					l.cc.LogWarning(
						fmt.Sprintf("xcloud.Loader load file fail, fileName:%s content:%s err:%s",
							name, string(b), errLoad.Error()))
				}
			}
			return true
		} else {
			l.cc.LogWarning(
				fmt.Sprintf("xcloud.Loader watch file update, but not changed. fileName:%s ", name))
		}
	} else {
		l.cc.LogWarning(
			fmt.Sprintf("xcloud.Loader get file content fail, fileName:%s err:%s",
				name, err.Error()))
	}
	return false
}

func (l *Loader) getLastModified(ctx context.Context, k string) (time.Time, error) {
	stat, err := l.cli.StatObject(ctx, l.cc.Bucket, k, minio.StatObjectOptions{})
	if err != nil {
		return time.Time{}, err
	}
	return stat.LastModified, nil
}
