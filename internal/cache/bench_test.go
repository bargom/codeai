package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkMemoryCache_Set(b *testing.B) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()
	value := []byte("benchmark-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(ctx, key, value, 0)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(ctx, key, []byte("value"), 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Get(ctx, key)
	}
}

func BenchmarkMemoryCache_SetGet(b *testing.B) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()
	value := []byte("benchmark-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(ctx, key, value, 0)
		cache.Get(ctx, key)
	}
}

func BenchmarkMemoryCache_JSON(b *testing.B) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	type Data struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	input := Data{ID: 1, Name: "test", Email: "test@example.com"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("json-key-%d", i)
		cache.SetJSON(ctx, key, input, 0)
		var output Data
		cache.GetJSON(ctx, key, &output)
	}
}

func BenchmarkMemoryCache_Concurrent(b *testing.B) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()
	value := []byte("benchmark-value")

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("concurrent-key-%d", i)
			cache.Set(ctx, key, value, 0)
			cache.Get(ctx, key)
			i++
		}
	})
}

func BenchmarkMemoryCache_MGet(b *testing.B) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	// Pre-populate cache
	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = fmt.Sprintf("mget-key-%d", i)
		cache.Set(ctx, keys[i], []byte("value"), 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.MGet(ctx, keys...)
	}
}

func BenchmarkMemoryCache_MSet(b *testing.B) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	items := make(map[string][]byte)
	for i := 0; i < 100; i++ {
		items[fmt.Sprintf("mset-key-%d", i)] = []byte("value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.MSet(ctx, items, 0)
	}
}

func BenchmarkMemoryCache_IncrDecr(b *testing.B) {
	cache := NewMemoryCache(Config{DefaultTTL: time.Minute})
	defer cache.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Incr(ctx, "counter")
	}
}

func BenchmarkMemoryCache_LRUEviction(b *testing.B) {
	cache := NewMemoryCache(Config{
		DefaultTTL: time.Minute,
		MaxMemory:  1024 * 1024, // 1MB
	})
	defer cache.Close()
	ctx := context.Background()
	value := make([]byte, 1024) // 1KB values

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("evict-key-%d", i)
		cache.Set(ctx, key, value, 0)
	}
}
