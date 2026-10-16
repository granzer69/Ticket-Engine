package booking

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"

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
	client        *redis.Client
	sha           string
	streamMaxLen  int64 // 0 = unlimited XADD
}

// NewAllocator loads the Lua script into Redis.
// Optional env TICKET_STREAM_MAXLEN enables approximate MAXLEN ~ on bookings.stream XADD.
func NewAllocator(ctx context.Context, client *redis.Client) (*Allocator, error) {
	sha, err := client.ScriptLoad(ctx, allocateLua).Result()
	if err != nil {
		return nil, fmt.Errorf("booking: script load: %w", err)
	}
	var maxlen int64
	if v := strings.TrimSpace(os.Getenv("TICKET_STREAM_MAXLEN")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("booking: invalid TICKET_STREAM_MAXLEN %q", v)
		}
		maxlen = n
	}
	return &Allocator{client: client, sha: sha, streamMaxLen: maxlen}, nil
}

// Allocate reserves one ticket for userID or returns a prior reservation (idempotent).
func (a *Allocator) Allocate(ctx context.Context, userID int) (Result, error) {
	if userID <= 0 {
		return Result{}, fmt.Errorf("booking: invalid user id %d", userID)
	}
	uid := strconv.Itoa(userID)
	args := []interface{}{uid}
	if a.streamMaxLen > 0 {
		args = append(args, strconv.FormatInt(a.streamMaxLen, 10))
	}
	raw, err := a.client.EvalSha(ctx, a.sha, []string{KeyQueue, KeyUserBooking, KeyStream}, args...).Result()
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

// TicketForUser returns the ticket id stored in Redis for userID, if any.
func TicketForUser(ctx context.Context, client *redis.Client, userID int) (int, bool, error) {
	raw, err := client.HGet(ctx, KeyUserBooking, strconv.Itoa(userID)).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	tid, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false, fmt.Errorf("booking: bad hash ticket id %q: %w", raw, err)
	}
	return tid, true, nil
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
