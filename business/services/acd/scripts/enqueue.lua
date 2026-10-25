-- enqueue.lua - 通话入队原子操作
-- 功能：ZADD active + HSET call:data + SADD working-ids
-- KEYS[1] = active queue key (vocara:acd:queue:active:{queueId})
-- KEYS[2] = call data key (vocara:acd:queue:call:{callId})
-- KEYS[3] = working ids set key (vocara:acd:queue:working-ids)
-- ARGV[1] = score (priority * 10^12 + eqTimestamp)
-- ARGV[2] = callId
-- ARGV[3] = caller
-- ARGV[4] = called
-- ARGV[5] = queueId
-- ARGV[6] = skillGroupId
-- ARGV[7] = priority
-- ARGV[8] = eqTime (unix timestamp in seconds)
-- ARGV[9] = tenantId
-- ARGV[10] = customerInfo (optional, can be empty string)
-- ARGV[11] = uui (optional, can be empty string)
-- ARGV[12] = maxQueueLen (0 = unlimited)

local active_key = KEYS[1]
local call_key = KEYS[2]
local working_key = KEYS[3]

local score = ARGV[1]
local call_id = ARGV[2]
local caller = ARGV[3]
local called = ARGV[4]
local queue_id = ARGV[5]
local skill_group_id = ARGV[6]
local priority = ARGV[7]
local eq_time = ARGV[8]
local tenant_id = ARGV[9]
local customer_info = ARGV[10]
local uui = ARGV[11]
local max_queue_len = tonumber(ARGV[12])

-- 检查是否已存在该通话（防止重复入队）
if redis.call('EXISTS', call_key) == 1 then
    return {'CALL_ALREADY_EXISTS'}
end

-- 检查队列长度限制
if max_queue_len > 0 then
    local current_len = redis.call('ZCARD', active_key)
    if current_len >= max_queue_len then
        return {'QUEUE_FULL'}
    end
end

-- 1. 将通话加入 active 队列（ZSET）
redis.call('ZADD', active_key, score, call_id)

-- 2. 保存通话详情（HASH）
redis.call('HSET', call_key,
    'callId', call_id,
    'caller', caller,
    'called', called,
    'queueId', queue_id,
    'skillGroupId', skill_group_id,
    'priority', priority,
    'currentPriority', priority,
    'status', 'waiting',
    'eqTime', eq_time,
    'tenantId', tenant_id,
    'customerInfo', customer_info,
    'uui', uui
)

-- 3. 标记队列为工作中
redis.call('SADD', working_key, queue_id)

-- 4. 返回结果
return {'ENQUEUED', score}
