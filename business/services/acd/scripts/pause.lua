-- pause.lua - 暂停通话分配
-- 功能：从 active 移除 + 加入 paused + 更新 call status
-- KEYS[1] = active queue key
-- KEYS[2] = paused queue key
-- KEYS[3] = call data key (vocara:acd:queue:call:{callId})
-- ARGV[1] = callId
-- ARGV[2] = pause timestamp
-- ARGV[3] = pause reason (optional)

local active_key = KEYS[1]
local paused_key = KEYS[2]
local call_key = KEYS[3]

local call_id = ARGV[1]
local pause_time = ARGV[2]
local pause_reason = ARGV[3]

-- 检查通话是否存在
if redis.call('EXISTS', call_key) == 0 then
    return {'CALL_NOT_FOUND'}
end

-- 检查当前状态
local status = redis.call('HGET', call_key, 'status')
if status == 'paused' then
    return {'ALREADY_PAUSED'}
end
if status ~= 'waiting' then
    return {'INVALID_STATUS'}
end

-- 1. 从 active 队列移除
local removed = redis.call('ZREM', active_key, call_id)
if removed == 0 then
    -- 不在 active 队列中，可能状态不一致，尝试修正
    return {'NOT_IN_ACTIVE_QUEUE'}
end

-- 2. 加入 paused 队列
redis.call('ZADD', paused_key, pause_time, call_id)

-- 3. 更新通话状态
redis.call('HSET', call_key, 'status', 'paused', 'pauseTime', pause_time)
if pause_reason and pause_reason ~= '' then
    redis.call('HSET', call_key, 'pauseReason', pause_reason)
end

return {'PAUSED'}
