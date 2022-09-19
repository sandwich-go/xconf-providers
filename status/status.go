package status

import (
	"sync"
)

type OnConfUpdate func(string, []byte)

var LastStatus = &status{
	conf: make(map[string][]byte),
}

type status struct {
	sync.RWMutex
	conf map[string][]byte
}
type FileContent struct {
	Name    string
	Content []byte
}

func (m *status) List() []*FileContent {
	m.RLock()
	defer m.RUnlock()
	ret := make([]*FileContent, 0, len(m.conf))
	for k, v := range m.conf {
		ret = append(ret, &FileContent{
			Name:    k,
			Content: v,
		})
	}
	return ret
}

func (m *status) UpdateConf(fileName string, content []byte) {
	m.Lock()
	defer m.Unlock()
	m.conf[fileName] = content
}
