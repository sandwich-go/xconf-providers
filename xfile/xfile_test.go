package xfile

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func TestFileLoader(t *testing.T) {
	l, err := New()
	if err != nil {
		t.Fatal(err)
	}
	fileName := "test.txt"
	defer func(name string) {
		_ = os.Remove(name)
	}(fileName)
	err = os.WriteFile(fileName, []byte("hello world"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 10)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		id := i
		l.Watch(context.Background(), fileName, func(loaderName string, confPath string, content []byte) error {
			defer wg.Done()
			fmt.Println(id, " watch file change:", string(content))
			return nil
		})
	}
	ch := make(chan struct{})
	ch2 := make(chan struct{})
	go func() {
		defer close(ch2)
		err = os.WriteFile(fileName, []byte("hello xconf"), 0666)
		if err != nil {
			fmt.Println("watch file changed")
			return
		}
		select {
		case <-ch:
			return
		case <-time.After(time.Second * 2):
			fmt.Println("can not watch file change after 2 seconds")
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
