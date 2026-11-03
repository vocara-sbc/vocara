-- dequeue.lua - 通话出队（挂机、转接、分配成功等）
-- 功能：从 active/paused 移除 + DEL call:data + 更新 stats
-- KEYS[1] = active queue key
-- KEYS[2] = paused queue key
-- KEYS[3] = call data key
-- KEYS[4] = stats key (vocara:acd:queue:stats:{queueId})
-- KEYS[5] = working ids set key
-- ARGV[1] = callId
-- ARGV[2] = dequeue reason: answered | abandoned | transferred | timeout | error
-- ARGV[3] = dequeue timestamp

local active_key = KEYS[1]
local paused_key = KEYS[2]
local call_key = KEYS[3]
local stats_key = KEYS[4]
local working_key = KEYS[5]

local call_id = ARGV[1]
local reason = ARGV[2]
local dequeue_time = tonumber(ARGV[3])

-- 检查通话是否存在
if redis.call('EXISTS', call_key) == 0 then
    return {'CALL_NOT_FOUND'}
end

-- 获取通话信息
local call_data = redis.call('HMGET', call_key, 'queueId', 'eqTime', 'status')
local queue_id = call_data[1]
local eq_time = tonumber(call_data[2])
local status = call_data[3]

-- 1. 从对应队列中移除
if status == 'waiting' then
    redis.call('ZREM', active_key, call_id)
elseif status == 'paused' then
    redis.call('ZREM', paused_key, call_id)
end

-- 2. 计算等待时长
local wait_time = 0
if eq_time and dequeue_time then
    wait_time = dequeue_time - eq_time
    if wait_time < 0 then wait_time = 0 end
end

-- 3. 更新统计数据
if reason == 'answered' then
    redis.call('HINCRBY', stats_key, 'answered_today', 1)

    -- 更新平均等待时间（增量计算）
    local answered = tonumber(redis.call('HGET', stats_key, 'answered_today') or '0')
    local avg_wait = tonumber(redis.call('HGET', stats_key, 'avg_wait_time') or '0')
    if answered > 0 then
        local new_avg = (avg_wait * (answered - 1) + wait_time) / answered
        redis.call('HSET', stats_key, 'avg_wait_time', string.format('%.2f', new_avg))
    end

    -- 通话中计数 +1
    redis.call('HINCRBY', stats_key, 'serving', 1)

    -- SLA 统计（20秒内接听算达标）
    if wait_time <= 20 then
        redis.call('HINCRBY', stats_key, 'sla_answered', 1)
    end
    local sla_answered = tonumber(redis.call('HGET', stats_key, 'sla_answered') or '0')
    local total_answered = tonumber(redis.call('HGET', stats_key, 'answered_today') or '0')
    if total_answered > 0 then
        local sla = (sla_answered / total_answered) * 100
        redis.call('HSET', stats_key, 'sla', string.format('%.2f', sla))
    end
elseif reason == 'abandoned' then
    redis.call('HINCRBY', stats_key, 'abandoned_today', 1)
elseif reason == 'timeout' then
    redis.call('HINCRBY', stats_key, 'timeout_today', 1)
elseif reason == 'transferred' then
    redis.call('HINCRBY', stats_key, 'transferred_today', 1)
end

-- 4. 删除通话详情（分配成功的情况在呼叫控制那边保留通话记录，这里只删 ACD 的排队信息）
-- 如果是 answered，保留 call info 一小段时间用于查询，这里先标记状态
if reason == 'answered' then
    redis.call('HSET', call_key, 'status', 'serving', 'dequeueTime', dequeue_time, 'dequeueReason', reason)
    -- 设置过期时间（1小时后自动删除）
    redis.call('EXPIRE', call_key, 3600)
else
    redis.call('DEL', call_key)
end

-- 5. 检查队列是否为空，更新 working-ids
local active_count = redis.call('ZCARD', active_key)
local paused_count = redis.call('ZCARD', paused_key)
if active_count == 0 and paused_count == 0 then
    -- 两个队列都空了，从 working-ids 移除
    redis.call('SREM', working_key, queue_id)
end

-- 6. 更新统计的 waiting 计数
redis.call('HSET', stats_key, 'waiting', active_count)
redis.call('HSET', stats_key, 'paused', paused_count)

return {'DEQUEUED', wait_time}
