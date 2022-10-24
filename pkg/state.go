package pkg

import "fmt"

type Persistence interface {
	Exists(string) (error, bool)
	Add(string, interface{}) error
	Rm(string) error
	Get(string, interface{}) error
}

var _ Persistence = &MemoryState{}

type MemoryState struct {
	state map[string]interface{}
}

func NewMemoryState() *MemoryState {
	return &MemoryState{
		state: make(map[string]interface{}),
	}
}

func (s *MemoryState) Exists(key string) (error, bool) {
	if ok := s.state[key]; ok != nil {
		return nil, true
	}
	return nil, false
}

func (s *MemoryState) Add(key string, value interface{}) error {
	s.state[key] = value
	return nil
}

func (s *MemoryState) Rm(key string) error {
	delete(s.state, key)
	return nil
}

func (s *MemoryState) Get(key string, value interface{}) error {
	if _, ok := s.Exists(key); ok {
		// skipping error is fine MemoryState
		value = s.state[key]
		return nil
	}
	return fmt.Errorf("Key does not exists")
}
