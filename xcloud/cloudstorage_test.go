package xcloud

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sandwich-go/minio-go"
)

var (
	bucket = "zhongtai"
)

func TestHuaweiCloudLoader(t *testing.T) {
	fileName := "/test/conf.ini"
	content := "hello cloud storage xconf"
	bucket = os.Getenv("RELEASE_HUAWEIRU_BUCKET")
	l, err := New(
		WithStorageType(StorageTypeHuaweiRu),
		WithAccessKey(os.Getenv("RELEASE_HUAWEIRU_KEY")),
		WithSecret(os.Getenv("RELEASE_HUAWEIRU_SECRET")),
		WithRegion("ru-moscow-1"),
		WithBucket(bucket))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		delObject(l.(*Loader).cli, fileName)
	}()
	putObject(l.(*Loader).cli, fileName, []byte(content))
}

func TestNewCloudLoader(t *testing.T) {
	fileName := "/test/conf.ini"
	content := "hello cloud storage xconf"
	l, err := New(
		WithStorageType(StorageTypeS3),
		WithAccessKey(os.Getenv("RELEASE_CLOUD_KEY")),
		WithSecret(os.Getenv("RELEASE_CLOUD_SECRET")),
		WithRegion("us-east-2"),
		WithBucket(bucket))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		delObject(l.(*Loader).cli, fileName)
	}()
	putObject(l.(*Loader).cli, fileName, []byte(content))
	time.Sleep(time.Millisecond * 10)

	ctxGet, cancelGet := context.WithTimeout(context.Background(), time.Second*3)
	get, err := l.Get(ctxGet, fileName)
	cancelGet()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("get conf", string(get))
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		id := i
		l.Watch(context.Background(), fileName, func(loaderName string, confPath string, cc []byte) error {
			if string(cc) != content {
				defer wg.Done()
			}
			fmt.Println(time.Now(), " id: ", id, " watch file change:", string(cc))
			return nil
		})
	}
	ch := make(chan struct{})
	ch2 := make(chan struct{})
	go func() {
		time.Sleep(time.Second * 2)
		defer close(ch2)
		putObject(l.(*Loader).cli, fileName, []byte("new content"))
		fmt.Println(time.Now(), " put object")
		if err != nil {
			fmt.Println("file changed")
			return
		}
		select {
		case <-ch:
			return
		case <-time.After(time.Second * 5):
			fmt.Println("can not watch file change after 5 seconds")
			os.Exit(1)
		}
	}()
	wg.Wait()
	close(ch)
	select {
	case <-ch2:
		return
	case <-time.After(time.Second):
		return
	}
}

func putObject(cli *minio.Client, name string, content []byte) {
	_, err := cli.PutObject(context.Background(), bucket, name, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{})
	if err != nil {
		log.Fatal(err)
	}
}
func delObject(cli *minio.Client, name string) {
	err := cli.RemoveObject(context.Background(), bucket, name, minio.RemoveObjectOptions{})
	if err != nil {
		log.Fatal(err)
	}
}
