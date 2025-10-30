package xvault

import (
	"log"

	"github.com/sandwich-go/xconf"
	"github.com/sandwich-go/xconf/kv"

	"github.com/sandwich-go/xconf-providers/status"
)

//go:generate optiongen --option_with_struct_name=false --option_return_previous=false
func OptionsOptionDeclareWithDefault() interface{} {
	return map[string]interface{}{
		"Address":         "",                                                    // Vault地址，例如：http://localhost:8200
		"Token":           "",                                                    // Token认证方式使用的token
		"Role":            "",                                                    // Kubernetes认证方式使用的role
		"ServiceAccount":  "/var/run/secrets/kubernetes.io/serviceaccount/token", // K8s ServiceAccount token路径
		"KubernetesMount": "kubernetes",                                          // Kubernetes认证挂载点
		"SecretMount":     "secret",                                              // secret引擎
		"AutoDetectEnv":   true,                                                  // 是否自动检测环境（K8s或本地）
		"KVOption":        []kv.Option(nil),
		"LogDebug":        xconf.LogFunc(func(s string) { log.Println("[  DEBUG] " + s) }),
		"LogWarning":      xconf.LogFunc(func(s string) { log.Println("[WARNING] " + s) }),
		"OnUpdate":        status.OnConfUpdate(status.LastStatus.UpdateConf),
	}
}
