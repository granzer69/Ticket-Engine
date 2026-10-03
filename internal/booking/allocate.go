package booking

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
)

//go:embed lua_allocate.lua
var allocateLua string

const (
	StatusNew      = 0
	StatusReplay   = 1
	StatusSoldOut  = 2
	KeyQueue       = "tickets"
	KeyUserBooking = "hash:user"
	KeyStream      = "bookings.stream"
)

// Result of an allocation attempt.
type Result struct {
	TicketID int
	Replay   bool
	SoldOut  bool
}

// Allocator performs atomic Redis ticket allocation.
type Allocator struct {
	client *redis.Client
	sha    string
}

// NewAllocator loads the Lua script into Redis.
func NewAllocator(ctx context.Context, client *redis.Client) (*Allocator, error) {
	sha, err := client.ScriptLoad(ctx, allocateLua).Result()
	if err != nil {
		return nil, fmt.Errorf("booking: script load: %w", err)
	}
	return &Allocator{client: client, sha: sha}, nil
}

// Allocate reserves one ticket for userID or returns a prior reservation (idempotent).
func (a *Allocator) Allocate(ctx context.Context, userID int) (Result, error) {
	if userID <= 0 {
		return Result{}, fmt.Errorf("booking: invalid user id %d", userID)
	}
	uid := strconv.Itoa(userID)
	raw, err := a.client.EvalSha(ctx, a.sha, []string{KeyQueue, KeyUserBooking, KeyStream}, uid).Result()
	if err != nil {
		return Result{}, fmt.Errorf("booking: eval: %w", err)
	}
	return parseAllocateResult(raw)
}

func parseAllocateResult(raw interface{}) (Result, error) {
	vals, ok := raw.([]interface{})
	if !ok || len(vals) != 2 {
		return Result{}, fmt.Errorf("booking: unexpected redis response %T", raw)
	}
	status, err := toInt(vals[0])
	if err != nil {
		return Result{}, err
	}
	ticketStr, _ := vals[1].(string)
	switch status {
	case StatusNew:
		tid, err := strconv.Atoi(ticketStr)
		if err != nil {
			return Result{}, fmt.Errorf("booking: bad ticket id %q: %w", ticketStr, err)
		}
		return Result{TicketID: tid, Replay: false}, nil
	case StatusReplay:
		tid, err := strconv.Atoi(ticketStr)
		if err != nil {
			return Result{}, fmt.Errorf("booking: bad replay ticket id %q: %w", ticketStr, err)
		}
		return Result{TicketID: tid, Replay: true}, nil
	case StatusSoldOut:
		return Result{SoldOut: true}, nil
	default:
		return Result{}, fmt.Errorf("booking: unknown status %d", status)
	}
}

func toInt(v interface{}) (int, error) {
	switch n := v.(type) {
	case int64:
		return int(n), nil
	case int:
		return n, nil
	default:
		return 0, fmt.Errorf("booking: expected int status, got %T", v)
	}
}
