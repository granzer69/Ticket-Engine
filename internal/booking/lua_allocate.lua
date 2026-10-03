-- Atomic ticket allocation with per-user idempotency.
-- KEYS[1] = ticket queue list
-- KEYS[2] = user booking hash (field: user id, value: ticket id)
-- ARGV[1] = user id string
--
-- Returns {status, ticket_id}
-- status: 0 = newly allocated, 1 = idempotent replay, 2 = sold out

local existing = redis.call('HGET', KEYS[2], ARGV[1])
if existing then
  return {1, existing}
end

local ticket = redis.call('LPOP', KEYS[1])
if not ticket then
  return {2, ''}
end

redis.call('HSET', KEYS[2], ARGV[1], ticket)
return {0, ticket}
