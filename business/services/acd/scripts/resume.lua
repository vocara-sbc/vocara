-- resume.lua - 恢复通话分配
-- 功能：从 paused 移除 + 加入 active + 更新 call status
-- KEYS[1] = active queue key
-- KEYS[2] = paused queue key
-- KEYS[3] = call data key
-- KEYS[4] = working ids set key
-- ARGV[1] = callId
-- ARGV[2] = resume timestamp (current time)

local active_key = KEYS[1]
local paused_key = KEYS[2]
local call_key = KEYS[3]
local working_key = KEYS[4]

local call_id = ARGV[1]
local resume_time = tonumber(ARGV[2])

-- 检查通话是否存在
if redis.call('EXISTS', call_key) == 0 then
    return {'CALL_NOT_FOUND'}
end

-- 检查当前状态
local status = redis.call('HGET', call_key, 'status')
if status == 'waiting' then
    return {'ALREADY_ACTIVE'}
end
if status ~= 'paused' then
    return {'INVALID_STATUS'}
end

-- 1. 从 paused 队列移除
local removed = redis.call('ZREM', paused_key, call_id)
if removed == 0 then
    return {'NOT_IN_PAUSED_QUEUE'}
end

-- 2. 计算新的 score（基于入队时间，保持原始排队位置）
local eq_time = redis.call('HGET', call_key, 'eqTime')
local current_priority = redis.call('HGET', call_key, 'currentPriority')
local new_score = current_priority * 10^12 + eq_time

-- 3. 加入 active 队列
redis.call('ZADD', active_key, new_score, call_id)

-- 4. 更新通话状态
redis.call('HSET', call_key, 'status', 'waiting', 'resumeTime', resume_time)

-- 5. 确保队列标记为工作中
local queue_id = redis.call('HGET', call_key, 'queueId')
redis.call('SADD', working_key, queue_id)

return {'RESUMED', new_score}
