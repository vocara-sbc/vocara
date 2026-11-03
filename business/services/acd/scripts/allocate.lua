-- allocate.lua - 窥视队首通话用于分配
-- 功能：ZRANGE active 0 0（peek 语义，不移除队首成员），返回队首通话详情
-- KEYS[1] = active queue key (vocara:acd:queue:active:{queueId})
-- KEYS[2] = paused queue key (vocara:acd:queue:paused:{queueId})
-- KEYS[3] = call data key prefix (vocara:acd:queue:call:)
-- ARGV[1] = current timestamp (for aging check)
-- ARGV[2] = aging cycle in seconds (0 = disabled)
-- ARGV[3] = max priority (e.g. 3 for VIP)
--
-- 返回：
--   {ok, callId, score, caller, called, priority, currentPriority, eqTime, ...}
--   或 {err = 'EMPTY_QUEUE'}

local active_key = KEYS[1]
local call_prefix = KEYS[3]
local now = tonumber(ARGV[1])
local aging_cycle = tonumber(ARGV[2])
local max_priority = tonumber(ARGV[3])

-- 循环查找第一个有效通话（跳过已失效的）
while true do
    -- 取出队首元素（score 最小 = 优先级最高 或 等待最久）
    local members = redis.call('ZRANGE', active_key, 0, 0, 'WITHSCORES')
    if #members == 0 then
        return {'EMPTY_QUEUE'}
    end

    local call_id = members[1]
    local score = tonumber(members[2])
    local call_key = call_prefix .. call_id

    -- 检查通话详情是否存在
    if redis.call('EXISTS', call_key) == 0 then
        -- 通话详情不存在，说明是脏数据，移除并继续
        redis.call('ZREM', active_key, call_id)
    else
        -- 获取通话详情
        local call_data = redis.call('HMGET', call_key,
            'caller', 'called', 'queueId', 'skillGroupId',
            'priority', 'currentPriority', 'eqTime',
            'tenantId', 'customerInfo', 'uui', 'status'
        )

        -- 检查状态是否是 waiting（防止状态不一致）
        if call_data[11] ~= 'waiting' then
            redis.call('ZREM', active_key, call_id)
        else
            -- 优先级老化处理
            local current_priority = tonumber(call_data[6])
            local eq_time = tonumber(call_data[7])
            local wait_seconds = now - eq_time

            if aging_cycle > 0 and wait_seconds > 0 and current_priority < max_priority then
                -- 计算应该提升到的优先级
                local aging_levels = math.floor(wait_seconds / aging_cycle)
                local base_priority = tonumber(call_data[5])
                local new_priority = base_priority + aging_levels
                if new_priority > max_priority then
                    new_priority = max_priority
                end

                -- 如果优先级有变化，更新 score 和 currentPriority
                if new_priority > current_priority then
                    local new_score = new_priority * 10^12 + eq_time
                    redis.call('ZADD', active_key, new_score, call_id)
                    redis.call('HSET', call_key, 'currentPriority', new_priority)

                    -- 优先级变化后重新排序，需要重新取队首
                    -- 因为这个通话可能不再是队首了
                    return {'PRIORITY_UPDATED_RETRY'}
                end
            end

            -- 返回队首通话信息（不从 active 队列移除，分配成功后再 dequeue）
            return {
                'ok',
                call_id,            -- 1
                score,              -- 2
                call_data[1],       -- 3 caller
                call_data[2],       -- 4 called
                call_data[3],       -- 5 queueId
                call_data[4],       -- 6 skillGroupId
                call_data[5],       -- 7 priority (base)
                call_data[6],       -- 8 currentPriority
                call_data[7],       -- 9 eqTime
                call_data[8],       -- 10 tenantId
                call_data[9],       -- 11 customerInfo
                call_data[10],      -- 12 uui
                wait_seconds        -- 13 waitSeconds
            }
        end
    end
end
