package xfile

import (
	"log"

	"github.com/sandwich-go/xconf"
	"github.com/sandwich-go/xconf/kv"

	"github.com/sandwich-go/xconf-providers/status"
)

//go:generate optiongen --option_with_struct_name=false --option_return_previous=false
func OptionsOptionDeclareWithDefault() interface{} {
	return map[string]interface{}{
		// annotation@PollingMode(comment="Docker运行时fs-event方式不可用时可强制使用polling模式")
		"PollingMode": false,
		"KVOption":    []kv.Option(nil),
		"LogDebug":    xconf.LogFunc(func(s string) { log.Println("[  DEBUG] " + s) }),
		"LogWarning":  xconf.LogFunc(func(s string) { log.Println("[WARNING] " + s) }),
		"OnUpdate":    status.OnConfUpdate(status.LastStatus.UpdateConf),
	}
}
