package chatstore

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
)

type ChatStore struct {
	mu   sync.Mutex
	path string
}

func NewChatStore(path string) *ChatStore {
	return &ChatStore{path: path}
}

func (s *ChatStore) Load() map[int64]bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	chats := make(map[int64]bool)

	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return chats
		}
		log.Printf("chatstore load: %v", err)
		return chats
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		id, err := strconv.ParseInt(scanner.Text(), 10, 64)
		if err != nil {
			continue
		}
		chats[id] = true
	}
	if err := scanner.Err(); err != nil {
		log.Printf("chatstore scan: %v", err)
	}

	log.Printf("Loaded %d chats from %s", len(chats), s.path)
	return chats
}

func (s *ChatStore) Save(chats map[int64]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Create(s.path)
	if err != nil {
		log.Printf("chatstore save: %v", err)
		return
	}
	defer f.Close()

	for id := range chats {
		fmt.Fprintln(f, id)
	}
}
