package xvault

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/sandwich-go/boost/xerror"
	"github.com/sandwich-go/boost/xpanic"
	"github.com/sandwich-go/xconf/kv"
)

const (
	LoaderName = "vault"
)

// New make vault kv.Loader
func New(opts ...Option) (p kv.Loader, err error) {
	opt := NewOptions(opts...)

	// 创建vault client配置
	config := vault.DefaultConfig()
	config.Address = opt.Address

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// 根据配置自动检测环境并认证
	if opt.AutoDetectEnv {
		if err := autoAuth(client, opt); err != nil {
			return nil, fmt.Errorf("failed to authenticate: %w", err)
		}
	} else {
		// 手动认证
		if opt.Token != "" {
			client.SetToken(opt.Token)
		}
	}

	x := &Loader{
		cc:           opt,
		client:       client,
		onChanged:    make(map[string][]kv.ContentChange),
		lastModified: make(map[string]int),
	}
	x.Common = kv.New(LoaderName, x, opt.KVOption...)

	go xpanic.AutoRecover(
		"xvault.worker",
		x.watchEvent,
		xpanic.WithAutoRecoverOptionOnRecover(func(tag string, reason interface{}) {
			x.cc.LogWarning(fmt.Sprintf("%s panic recover reason:%v", tag, reason))
		}))

	return x, nil
}

// autoAuth 自动检测环境并进行认证
func autoAuth(client *vault.Client, opt *Options) error {
	// 检测是否在K8s环境中
	if isRunningInKubernetes() {
		opt.LogDebug("Detected Kubernetes environment, using kubernetes auth method")
		return kubernetesAuth(client, opt)
	}

	// 本地开发环境，使用token认证
	opt.LogDebug("Detected local environment, using token auth method")
	if opt.Token == "" {
		return fmt.Errorf("token is required for local development")
	}
	client.SetToken(opt.Token)
	return nil
}

// isRunningInKubernetes 检测是否在Kubernetes环境中运行
func isRunningInKubernetes() bool {
	// 检查K8s Service Account token文件是否存在
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		return true
	}
	// 检查环境变量
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}
	return false
}

// kubernetesAuth 使用Kubernetes认证方式
func kubernetesAuth(client *vault.Client, opt *Options) error {
	// 读取Service Account token
	jwtBytes, err := os.ReadFile(opt.ServiceAccount)
	if err != nil {
		return fmt.Errorf("failed to read service account token: %w", err)
	}
	jwt := string(jwtBytes)

	// 构造kubernetes认证请求
	data := map[string]interface{}{
		"jwt":  jwt,
		"role": opt.Role,
	}

	// 发送认证请求
	path := fmt.Sprintf("auth/%s/login", opt.KubernetesMount)
	secret, err := client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("kubernetes auth failed: %w", err)
	}

	if secret == nil || secret.Auth == nil || secret.Auth.ClientToken == "" {
		return fmt.Errorf("kubernetes auth returned no token")
	}

	// 设置token
	client.SetToken(secret.Auth.ClientToken)
	opt.LogDebug(fmt.Sprintf("Kubernetes auth successful, token TTL: %v", secret.Auth.LeaseDuration))

	return nil
}

// Loader vault Loader
type Loader struct {
	client *vault.Client
	*kv.Common
	mutex        sync.Mutex
	onChanged    map[string][]kv.ContentChange
	lastModified map[string]int
	cc           *Options
}

func (l *Loader) CloseImplement(ctx context.Context) error {
	return nil
}

// GetImplement 从Vault获取配置
func (l *Loader) GetImplement(ctx context.Context, confPath string) ([]byte, error) {
	secretPath, key, err := getPathAndKey(confPath)
	if err != nil {
		return nil, err
	}
	secret, err := l.client.KVv2(l.cc.SecretMount).Get(ctx, secretPath)
	if err != nil {
		return nil, err
	}
	if val, ok := secret.Data[key]; ok {
		switch val := val.(type) {
		case string:
			return []byte(val), nil
		case []byte:
			return val, nil
		default:
			return []byte(fmt.Sprintf("%v", val)), nil
		}
	}

	return nil, xerror.NewText("not found key:%s", key)
}

// WatchImplement 监听配置变化
func (l *Loader) WatchImplement(ctx context.Context, confPath string, onContentChange kv.ContentChange) {
	l.mutex.Lock()
	if len(l.onChanged[confPath]) == 0 {
		if version, err := l.getVersion(ctx, confPath); err == nil {
			l.lastModified[confPath] = version
		}
	}
	l.onChanged[confPath] = append(l.onChanged[confPath], onContentChange)
	l.mutex.Unlock()
}

func (l *Loader) watchEvent() {
	ticker := time.NewTicker(time.Second * 5) // Vault检查间隔较长，避免频繁请求
	defer ticker.Stop()

	for {
		select {
		case <-l.Done:
			return
		case <-ticker.C:
			l.checkChanges()
		}
	}
}

func (l *Loader) checkChanges() {
	l.mutex.Lock()
	paths := make([]string, 0, len(l.onChanged))
	for k := range l.onChanged {
		paths = append(paths, k)
	}
	l.mutex.Unlock()

	for _, path := range paths {
		select {
		case <-l.Done:
			return
		default:
		}

		l.mutex.Lock()
		lastVersion, ok := l.lastModified[path]
		l.mutex.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		currentVersion, err := l.getVersion(ctx, path)
		cancel()

		if err != nil {
			l.cc.LogWarning(fmt.Sprintf("xvault.Loader get version fail, path:%s err:%s", path, err.Error()))
			continue
		}

		if !ok || currentVersion > lastVersion {
			if l.fileChange(path) {
				l.mutex.Lock()
				l.lastModified[path] = currentVersion
				l.mutex.Unlock()
			}
		}
	}
}

func (l *Loader) fileChange(path string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	b, err := l.Get(ctx, path)
	if err != nil {
		l.cc.LogWarning(fmt.Sprintf("xvault.Loader get content fail, path:%s err:%s", path, err.Error()))
		return false
	}

	if l.IsChanged(path, b) {
		l.mutex.Lock()
		callbacks := l.onChanged[path]
		l.mutex.Unlock()

		for _, callback := range callbacks {
			if errLoad := callback(LoaderName, path, b); errLoad == nil {
				l.cc.LogDebug(fmt.Sprintf("xvault.Loader watch config update succ: %s", path))
				l.cc.OnUpdate(path, b)
			} else {
				l.cc.LogWarning(fmt.Sprintf("xvault.Loader load fail, path:%s err:%s", path, errLoad.Error()))
			}
		}
		return true
	} else {
		l.cc.OnUpdate(path, b)
		l.cc.LogWarning(fmt.Sprintf("xvault.Loader watch update, but not changed. path:%s", path))
	}
	return true
}

// getVersion 获取配置版本号
func (l *Loader) getVersion(ctx context.Context, confPath string) (int, error) {
	secretPath, key, err := getPathAndKey(confPath)
	if err != nil {
		return 0, err
	}
	meta, err := l.client.KVv2(l.cc.SecretMount).GetMetadata(ctx, secretPath)
	if err != nil {
		return 0, err
	}
	if val, ok := meta.Versions[key]; ok {
		return val.Version, nil
	}
	return 0, nil
}

func getPathAndKey(fullPath string) (string, string, error) {
	fullPath = strings.TrimSpace(fullPath)
	if fullPath == "" {
		return "", "", xerror.NewText("empty path")
	}

	dir := filepath.Dir(fullPath)
	file := filepath.Base(fullPath)

	// 校验
	if file == "." || file == "" {
		return "", "", xerror.NewText("invalid path: missing filename:%s", fullPath)
	}

	return dir, file, nil
}
