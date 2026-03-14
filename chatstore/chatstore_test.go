package chatstore

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func tmpStore(t *testing.T) *ChatStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "chats.txt")
	return NewChatStore(path)
}

func TestChatStore_LoadEmpty(t *testing.T) {
	s := tmpStore(t)
	chats := s.Load()
	if len(chats) != 0 {
		t.Errorf("expected empty map, got %d entries", len(chats))
	}
}

func TestChatStore_SaveAndLoad(t *testing.T) {
	s := tmpStore(t)

	chats := map[int64]bool{123: true, -456: true, 789: true}
	s.Save(chats)

	loaded := s.Load()
	if len(loaded) != 3 {
		t.Fatalf("expected 3 chats, got %d", len(loaded))
	}
	for id := range chats {
		if !loaded[id] {
			t.Errorf("missing chat %d after load", id)
		}
	}
}

func TestChatStore_OverwritesOnSave(t *testing.T) {
	s := tmpStore(t)

	s.Save(map[int64]bool{1: true, 2: true, 3: true})
	s.Save(map[int64]bool{4: true})

	loaded := s.Load()
	if len(loaded) != 1 {
		t.Fatalf("expected 1 chat after overwrite, got %d", len(loaded))
	}
	if !loaded[4] {
		t.Error("expected chat 4")
	}
}

func TestChatStore_SkipsInvalidLines(t *testing.T) {
	s := tmpStore(t)

	os.WriteFile(s.path, []byte("123\ninvalid\n456\n\n"), 0644)

	loaded := s.Load()
	if len(loaded) != 2 {
		t.Fatalf("expected 2 valid chats, got %d", len(loaded))
	}
}

func TestChatStore_ConcurrentLoadSave(t *testing.T) {
	s := tmpStore(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		id := int64(i)
		go func() {
			defer wg.Done()
			s.Save(map[int64]bool{id: true})
		}()
		go func() {
			defer wg.Done()
			s.Load()
		}()
	}
	wg.Wait()
}

func TestChatStore_EmptyFile(t *testing.T) {
	s := tmpStore(t)
	os.WriteFile(s.path, []byte(""), 0644)

	loaded := s.Load()
	if len(loaded) != 0 {
		t.Errorf("expected 0 chats from empty file, got %d", len(loaded))
	}
}

func TestChatStore_WhitespaceOnlyFile(t *testing.T) {
	s := tmpStore(t)
	os.WriteFile(s.path, []byte("   \n\t\n  \n"), 0644)

	loaded := s.Load()
	if len(loaded) != 0 {
		t.Errorf("expected 0 chats from whitespace file, got %d", len(loaded))
	}
}

func TestChatStore_UnreadableFile(t *testing.T) {
	s := tmpStore(t)
	os.WriteFile(s.path, []byte("123\n456\n"), 0644)

	// Make file unreadable
	os.Chmod(s.path, 0000)
	t.Cleanup(func() { os.Chmod(s.path, 0644) })

	loaded := s.Load()
	if len(loaded) != 0 {
		t.Errorf("expected 0 chats from unreadable file, got %d", len(loaded))
	}
}
