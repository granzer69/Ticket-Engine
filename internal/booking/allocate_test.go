package booking

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func newTestAllocator(t *testing.T, ticketIDs ...string) (*Allocator, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	for i := len(ticketIDs) - 1; i >= 0; i-- {
		if err := client.LPush(context.Background(), KeyQueue, ticketIDs[i]).Err(); err != nil {
			t.Fatalf("lpush: %v", err)
		}
	}
	alloc, err := NewAllocator(context.Background(), client)
	if err != nil {
		t.Fatalf("allocator: %v", err)
	}
	return alloc, mr
}

func TestAllocateNewAndIdempotentReplay(t *testing.T) {
	alloc, mr := newTestAllocator(t, "101", "102")
	ctx := context.Background()

	first, err := alloc.Allocate(ctx, 42)
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if first.Replay || first.SoldOut || first.TicketID != 101 {
		t.Fatalf("first: %+v", first)
	}

	replay, err := alloc.Allocate(ctx, 42)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !replay.Replay || replay.TicketID != 101 {
		t.Fatalf("replay: %+v", replay)
	}

	other, err := alloc.Allocate(ctx, 43)
	if err != nil || other.TicketID != 102 || other.Replay {
		t.Fatalf("other: %+v err=%v", other, err)
	}

	if mr.Exists(KeyUserBooking) == false {
		t.Fatal("expected user hash")
	}
}

func TestAllocateSoldOut(t *testing.T) {
	alloc, _ := newTestAllocator(t, "1")
	ctx := context.Background()
	if _, err := alloc.Allocate(ctx, 1); err != nil {
		t.Fatalf("first: %v", err)
	}
	out, err := alloc.Allocate(ctx, 2)
	if err != nil {
		t.Fatalf("sold out: %v", err)
	}
	if !out.SoldOut {
		t.Fatalf("expected sold out, got %+v", out)
	}
}

func TestAllocateInvalidUser(t *testing.T) {
	alloc, _ := newTestAllocator(t, "1")
	_, err := alloc.Allocate(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error for user 0")
	}
}

func TestConcurrentUsersUniqueTickets(t *testing.T) {
	const n = 50
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	for i := n; i >= 1; i-- {
		if err := client.LPush(ctx, KeyQueue, i).Err(); err != nil {
			t.Fatalf("lpush: %v", err)
		}
	}
	alloc, err := NewAllocator(ctx, client)
	if err != nil {
		t.Fatalf("allocator: %v", err)
	}

	var wg sync.WaitGroup
	seen := make(map[int]int)
	var mu sync.Mutex
	errCh := make(chan error, n)

	for uid := 1; uid <= n; uid++ {
		wg.Add(1)
		go func(user int) {
			defer wg.Done()
			res, err := alloc.Allocate(ctx, user)
			if err != nil {
				errCh <- err
				return
			}
			if res.SoldOut || res.Replay {
				errCh <- fmt.Errorf("user %d unexpected %+v", user, res)
				return
			}
			mu.Lock()
			if prev, ok := seen[res.TicketID]; ok {
				errCh <- fmt.Errorf("ticket %d assigned to %d and %d", res.TicketID, prev, user)
			}
			seen[res.TicketID] = user
			mu.Unlock()
		}(uid)
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		if e != nil {
			t.Fatal(e)
		}
	}
	if len(seen) != n {
		t.Fatalf("expected %d tickets, got %d", n, len(seen))
	}
}

func TestAllocateOneUserOneTicket(t *testing.T) {
	alloc, mr := newTestAllocator(t, "201", "202")
	ctx := context.Background()

	first, err := alloc.Allocate(ctx, 99)
	if err != nil || first.Replay || first.TicketID != 201 {
		t.Fatalf("first: %+v err=%v", first, err)
	}
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	remaining, err := client.LLen(ctx, KeyQueue).Result()
	if err != nil {
		t.Fatalf("llen: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("expected 1 ticket left after first allocate, got %d", remaining)
	}

	replay, err := alloc.Allocate(ctx, 99)
	if err != nil || !replay.Replay || replay.TicketID != 201 {
		t.Fatalf("replay: %+v err=%v", replay, err)
	}
	remaining, err = client.LLen(ctx, KeyQueue).Result()
	if err != nil {
		t.Fatalf("llen after replay: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("replay must not consume queue; expected 1 left, got %d", remaining)
	}
}

func TestAllocateOneTicketOneWinner(t *testing.T) {
	alloc, _ := newTestAllocator(t, "301")
	ctx := context.Background()

	winner, err := alloc.Allocate(ctx, 1)
	if err != nil || winner.SoldOut || winner.TicketID != 301 {
		t.Fatalf("winner: %+v err=%v", winner, err)
	}
	loser, err := alloc.Allocate(ctx, 2)
	if err != nil || !loser.SoldOut {
		t.Fatalf("second user: %+v err=%v", loser, err)
	}
}

func TestConcurrentIdempotentSameUser(t *testing.T) {
	alloc, _ := newTestAllocator(t, "99")
	ctx := context.Background()
	const workers = 20
	var wg sync.WaitGroup
	ids := make([]int, workers)
	errCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			res, err := alloc.Allocate(ctx, 7)
			if err != nil {
				errCh <- err
				return
			}
			if res.SoldOut {
				errCh <- fmt.Errorf("worker %d sold out", idx)
				return
			}
			ids[idx] = res.TicketID
		}(i)
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		if e != nil {
			t.Fatal(e)
		}
	}
	first := ids[0]
	for _, id := range ids {
		if id != first {
			t.Fatalf("idempotent mismatch: want %d got %v", first, ids)
		}
	}
}
