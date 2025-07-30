package xcloud

import (
	"errors"
	"fmt"
	"log"

	"github.com/sandwich-go/xconf"
	"github.com/sandwich-go/xconf/kv"

	"github.com/sandwich-go/xconf-providers/status"
)

type StorageType string

const (
	StorageTypeNoop     StorageType = "noop"
	StorageTypeS3       StorageType = "s3"
	StorageTypeQCloud   StorageType = "qcloud"
	StorageTypeGCS      StorageType = "gcs"
	StorageTypeHuaweiRu StorageType = "huaweiru"
)

//go:generate optiongen --option_with_struct_name=false --option_return_previous=false
func OptionsOptionDeclareWithDefault() interface{} {
	return map[string]interface{}{
		"StorageType": StorageType(StorageTypeNoop),
		"AccessKey":   "",
		"Secret":      "",
		"Region":      "",
		"Bucket":      "",
		"KVOption":    []kv.Option(nil),
		"LogDebug":    xconf.LogFunc(func(s string) { log.Println("[  DEBUG] " + s) }),
		"LogWarning":  xconf.LogFunc(func(s string) { log.Println("[WARNING] " + s) }),
		"OnUpdate":    status.OnConfUpdate(status.LastStatus.UpdateConf),
	}
}

type genEndpointFunc func(options *Options) (string, error)

var type2GenEndpointFunc = map[StorageType]genEndpointFunc{
	StorageTypeNoop: func(options *Options) (string, error) {
		return "", errors.New("must set storage type")
	},
	StorageTypeS3:       genS3Endpoint,
	StorageTypeQCloud:   genQCloudEndpoint,
	StorageTypeGCS:      genGCSEndpoint,
	StorageTypeHuaweiRu: genHuaweiRuEndpoint,
}

func genS3Endpoint(options *Options) (string, error) {
	if options.Region == "" {
		return "", errors.New("region is must")
	}
	return fmt.Sprintf("s3.dualstack.%s.amazonaws.com", options.Region), nil
}

func genQCloudEndpoint(options *Options) (string, error) {
	if options.Region == "" {
		return "cos.ap-beijing.myqcloud.com", nil
	}
	return fmt.Sprintf("cos.%s.myqcloud.com", options.Region), nil
}

func genGCSEndpoint(options *Options) (string, error) {
	return "storage.googleapis.com", nil
}

func genHuaweiRuEndpoint(options *Options) (string, error) {
	return "obs.ru-moscow-1.hc.sbercloud.ru", nil
}
