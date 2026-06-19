package cache

import (
	"sync"
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c := New(100*time.Millisecond, 0)
	defer c.Stop()

	c.Set("key1", "value1")

	got := c.Get("key1")
	if got != "value1" {
		t.Errorf("expected 'value1', got %v", got)
	}
}

func TestCacheExpired(t *testing.T) {
	c := New(50*time.Millisecond, 0)
	defer c.Stop()

	c.Set("key1", "value1")
	time.Sleep(100 * time.Millisecond)

	got := c.Get("key1")
	if got != nil {
		t.Errorf("expected nil for expired item, got %v", got)
	}
}

func TestCacheNotFound(t *testing.T) {
	c := New(time.Minute, 0)
	defer c.Stop()

	got := c.Get("nonexistent")
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestCacheDelete(t *testing.T) {
	c := New(time.Minute, 0)
	defer c.Stop()

	c.Set("key1", "value1")
	c.Delete("key1")

	got := c.Get("key1")
	if got != nil {
		t.Errorf("expected nil after delete, got %v", got)
	}
}

func TestCacheClear(t *testing.T) {
	c := New(time.Minute, 0)
	defer c.Stop()

	c.Set("a", 1)
	c.Set("b", 2)
	c.Clear()

	if c.Len() != 0 {
		t.Errorf("expected 0 after clear, got %d", c.Len())
	}
}

func TestCacheConcurrency(t *testing.T) {
	c := New(time.Minute, 0)
	defer c.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Set("key", n)
			c.Get("key")
			c.Delete("key")
		}(i)
	}
	wg.Wait()
}

func TestCacheLen(t *testing.T) {
	c := New(time.Minute, 0)
	defer c.Stop()

	if c.Len() != 0 {
		t.Errorf("expected 0, got %d", c.Len())
	}

	c.Set("a", 1)
	if c.Len() != 1 {
		t.Errorf("expected 1, got %d", c.Len())
	}

	c.Set("b", 2)
	if c.Len() != 2 {
		t.Errorf("expected 2, got %d", c.Len())
	}
}

func TestCacheSetWithTTL(t *testing.T) {
	c := New(time.Minute, 0)
	defer c.Stop()

	c.SetWithTTL("key1", "value1", 50*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	got := c.Get("key1")
	if got != nil {
		t.Errorf("expected nil for expired item, got %v", got)
	}
}
