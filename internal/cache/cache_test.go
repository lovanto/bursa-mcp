package cache

import (
	"context"
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	ctx := context.Background()
	c, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer c.Close()

	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok || string(got) != "v" {
		t.Fatalf("Get = %q, %v; want v, true", got, ok)
	}
}

func TestCacheMiss(t *testing.T) {
	ctx := context.Background()
	c, _ := Open(":memory:")
	defer c.Close()

	if _, ok, err := c.Get(ctx, "absent"); err != nil || ok {
		t.Fatalf("Get(absent) = ok %v, err %v; want false, nil", ok, err)
	}
}

func TestCacheExpiry(t *testing.T) {
	ctx := context.Background()
	c, _ := Open(":memory:")
	defer c.Close()

	// Negative TTL via a tiny duration that is already in the past by read time.
	if err := c.Set(ctx, "k", []byte("v"), time.Nanosecond); err != nil {
		t.Fatalf("Set: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Fatal("expected expired entry to miss")
	}
}

func TestCacheOverwrite(t *testing.T) {
	ctx := context.Background()
	c, _ := Open(":memory:")
	defer c.Close()

	_ = c.Set(ctx, "k", []byte("old"), time.Minute)
	_ = c.Set(ctx, "k", []byte("new"), time.Minute)
	got, ok, _ := c.Get(ctx, "k")
	if !ok || string(got) != "new" {
		t.Fatalf("Get = %q; want new", got)
	}
}
