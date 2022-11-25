package xfile

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/docker/docker/pkg/filenotify"
	"github.com/sandwich-go/boost/paniccatcher"
	"github.com/sandwich-go/xconf/kv"
)

const (
	LoaderName = "file"
)

// Loader file Loader
type Loader struct {
	cc        *Options
	watcher   filenotify.FileWatcher
	watchFile sync.Map
	*kv.Common
	onChanged map[string][]kv.ContentChange
	mutex     sync.Mutex
}

// New new file Loader
func New(opts ...Option) (p kv.Loader, err error) {
	opt := NewOptions(opts...)
	var watcher filenotify.FileWatcher
	if opt.PollingMode {
		watcher = filenotify.NewPollingWatcher()
	} else {
		if wh, er := filenotify.NewEventWatcher(); er != nil {
			opt.LogWarning(fmt.Sprintf("xfile.Loader new event watcher fail, new polling watcher instead. err:%s", er.Error()))
			watcher = filenotify.NewPollingWatcher()
		} else {
			watcher = wh
		}
	}

	x := &Loader{
		cc:        opt,
		watcher:   watcher,
		onChanged: make(map[string][]kv.ContentChange),
	}
	x.Common = kv.New(LoaderName, x, opt.KVOption...)
	go paniccatcher.AutoRecover(
		"xfile.worker",
		x.watchEvent,
		paniccatcher.WithAutoRecoverOptionOnRecover(func(tag string, reason interface{}) {
			x.cc.LogWarning(fmt.Sprintf("%s panic recover reason:%v", tag, reason))
		}))
	return x, nil
}

// CloseImplement 实现common.loaderImplement.CloseImplement
func (p *Loader) CloseImplement(ctx context.Context) error { return p.watcher.Close() }

// GetImplement 实现common.loaderImplement.GetImplement
func (p *Loader) GetImplement(ctx context.Context, confPath string) ([]byte, error) {
	return ioutil.ReadFile(confPath)
}

// WatchImplement 实现common.loaderImplement.WatchImplement
func (p *Loader) WatchImplement(ctx context.Context, confPath string, onContentChange kv.ContentChange) {
	if err := p.watcher.Add(confPath); err != nil {
		if p.CC.OnWatchError != nil {
			p.CC.OnWatchError(LoaderName, confPath, err)
		}
	}
	// PollingWatcher 返回的Name是 文件名
	p.watchFile.Store(filepath.Base(confPath), &fileInfo{
		path: confPath,
	})
	// EventWatcher 返回的的Name是 全路径
	p.watchFile.Store(confPath, &fileInfo{
		path: confPath,
	})
	p.mutex.Lock()
	p.onChanged[confPath] = append(p.onChanged[confPath], onContentChange)
	p.mutex.Unlock()
}

type fileInfo struct {
	path string
}

func (p *Loader) watchEvent() {
	for {
		select {
		case <-p.Done:
			return
		case event, ok := <-p.watcher.Events():
			if !ok {
				return
			}
			p.cc.LogDebug(fmt.Sprintf("xfile.Loader watch event: %s", event.String()))
			if f, ok := p.watchFile.Load(event.Name); ok {
				if info, ok2 := f.(*fileInfo); ok2 {
					p.fileChange(context.Background(), info.path)
				}
			}
		case err := <-p.watcher.Errors():
			select {
			case <-p.Done:
				return
			default:
			}
			if p.CC.OnWatchError != nil {
				p.CC.OnWatchError(LoaderName, "", err)
			}
		}
	}
}

func (p *Loader) fileChange(ctx context.Context, name string) {
	if b, err := p.Get(ctx, name); err == nil {
		if p.IsChanged(name, b) {
			p.mutex.Lock()
			for _, callback := range p.onChanged[name] {
				if errLoad := callback(LoaderName, name, b); errLoad == nil {
					p.cc.LogDebug(fmt.Sprintf("xfile.Loader watch config update succ: %s", name))
					p.cc.OnUpdate(name, b)
				} else {
					p.cc.LogWarning(
						fmt.Sprintf("xfile.Loader load file fail, fileName:%s content:%s err:%s",
							name, string(b), errLoad.Error()))
				}
			}
			p.mutex.Unlock()
		} else {
			// 没有变化的文件也要同步更新status，检查时跳过
			p.cc.OnUpdate(name, b)
			p.cc.LogWarning(
				fmt.Sprintf("xfile.Loader watch file update, but not changed. fileName:%s ", name))
		}
	} else {
		p.cc.LogWarning(
			fmt.Sprintf("xfile.Loader get file content fail, fileName:%s err:%s",
				name, err.Error()))
	}
}
