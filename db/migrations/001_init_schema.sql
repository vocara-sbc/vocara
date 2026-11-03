-- ============================================================
-- Vocara Call Center System - Database initialization script (baseline)
-- Database: MySQL 8.0+
-- Naming conventions:
--   1. Table names use a tb_ prefix + business semantics (tb_inbound_route, tb_line, etc.);
--   2. Primary keys are uniformly id (bigint unsigned auto-increment); primary key references are uniformly xxx_id (bigint unsigned);
--   3. Agent-id / number-like strings are uniformly xxx_number (agent_number/dn_number/did); code-like fields are uniformly xxx_code;
--   4. Time fields are uniformly created_at / updated_at (datetime(3) DEFAULT NULL);
--   5. Tenant fields are uniformly tenant_id (bigint unsigned);
--   6. Status fields are uniformly status (1-enabled 0-disabled); special state machines are commented separately;
--   7. Fields stay consistent with the GORM models (services/admin, services/agentcenter).
-- ============================================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================================
-- 1. Tenant table
-- ============================================================
DROP TABLE IF EXISTS `tb_tenant`;
CREATE TABLE `tb_tenant` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Tenant ID',
  `tenant_name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Tenant name',
  `tenant_code` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Tenant code',
  `contact_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Contact name',
  `contact_phone` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Contact phone',
  `contact_email` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Contact email',
  `address` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Address',
  `status` bigint NOT NULL DEFAULT '1' COMMENT 'Status: 1-enabled 0-disabled',
  `expire_time` datetime(3) DEFAULT NULL COMMENT 'Expiration time',
  `max_agents` bigint NOT NULL DEFAULT '0' COMMENT 'Max agents',
  `max_lines` bigint NOT NULL DEFAULT '0' COMMENT 'Max lines',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Remark',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_code` (`tenant_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Tenant table';

-- ============================================================
-- 2. Agent table (includes admin console login accounts; role=admin/supervisor can log in to the admin console)
-- ============================================================
DROP TABLE IF EXISTS `tb_agent`;
CREATE TABLE `tb_agent` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Agent ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  `agent_number` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Agent number (login account)',
  `agent_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Agent name',
  `password` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Login password (bcrypt hash)',
  `mobile` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Mobile number',
  `email` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Email',
  `avatar` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Avatar URL',
  `gender` bigint DEFAULT '0' COMMENT 'Gender: 0-unknown 1-male 2-female',
  `dn_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Bound extension number',
  `role` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT 'agent' COMMENT 'Role: admin-administrator supervisor-supervisor agent-agent',
  `status` bigint NOT NULL DEFAULT '1' COMMENT 'Status: 1-enabled 0-disabled',
  `work_status` bigint NOT NULL DEFAULT '0' COMMENT 'Work status: 0-offline 1-idle 2-busy 3-in-call 4-resting 5-after-call work',
  `group_id` bigint unsigned DEFAULT NULL COMMENT 'Owning agent group ID',
  `skill_levels` text COLLATE utf8mb4_unicode_ci COMMENT 'Skill levels (JSON)',
  `last_login_at` datetime(3) DEFAULT NULL COMMENT 'Last login time',
  `last_login_ip` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Last login IP',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Remark',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_agent_number` (`tenant_id`,`agent_number`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent table';

-- ============================================================
-- 3. Organization table
-- ============================================================
DROP TABLE IF EXISTS `tb_organization`;
CREATE TABLE `tb_organization` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Organization ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID',
  `org_name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Organization/department name',
  `org_code` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Organization code',
  `parent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Parent organization ID (0 = root)',
  `level` bigint NOT NULL DEFAULT '1' COMMENT 'Level (root = 1)',
  `path` varchar(1000) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Ancestor ID path, comma-separated',
  `leader_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Leader ID',
  `leader_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Leader name',
  `sort` bigint NOT NULL DEFAULT '0' COMMENT 'Sort order among siblings',
  `status` bigint NOT NULL DEFAULT '1' COMMENT 'Status: 1-enabled 0-disabled',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Remark',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_tenant_org` (`tenant_id`,`org_code`),
  KEY `idx_parent_id` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Organization table';

-- ============================================================
-- 4. Skill group table
-- ============================================================
DROP TABLE IF EXISTS `tb_skill_group`;
CREATE TABLE `tb_skill_group` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Skill group ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  `skill_code` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Skill group code',
  `skill_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Skill group name',
  `skill_type` bigint DEFAULT '1' COMMENT 'Skill type: 1-inbound 2-outbound 3-mixed',
  `status` bigint NOT NULL DEFAULT '1' COMMENT 'Status: 1-enabled 0-disabled',
  `priority` bigint DEFAULT '0' COMMENT 'Priority',
  `agent_count` bigint DEFAULT '0' COMMENT 'Agent count',
  `description` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Description',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Remark',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_skill_code` (`skill_code`),
  KEY `idx_tenant_id` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Skill group table';

-- ============================================================
-- 6. Agent-skill group relation table (shared by admin and agentcenter)
-- ============================================================
DROP TABLE IF EXISTS `tb_agent_skill`;
CREATE TABLE `tb_agent_skill` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned NOT NULL COMMENT 'Tenant ID',
  `skill_group_id` bigint unsigned NOT NULL COMMENT 'Skill group ID',
  `agent_id` bigint unsigned NOT NULL COMMENT 'Agent ID',
  `agent_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Agent number',
  `agent_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Agent name',
  `skill_level` bigint DEFAULT '5' COMMENT 'Skill level 1-10',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_skill` (`agent_id`,`skill_group_id`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_skill_agent` (`skill_group_id`,`agent_id`),
  KEY `idx_agent_number` (`agent_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent-skill group relation table';

-- ============================================================
-- 7. Extension table (SIP registration credentials; FS dynamic directory data source)
-- ============================================================
DROP TABLE IF EXISTS `tb_dn`;
CREATE TABLE `tb_dn` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Extension ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  `dn_number` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Extension number',
  `dn_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Extension name',
  `password` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'SIP registration password',
  `dn_type` bigint DEFAULT '1' COMMENT 'Extension type: 1-agent extension 2-general extension',
  `agent_id` bigint unsigned DEFAULT NULL COMMENT 'Bound agent ID',
  `agent_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Bound agent number',
  `status` bigint NOT NULL DEFAULT '1' COMMENT 'Status: 1-enabled 0-disabled',
  `register_status` bigint DEFAULT '0' COMMENT 'Registration status: 0-unregistered 1-registered',
  `ip_address` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Registration IP',
  `user_agent` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Registration user-agent',
  `last_register_at` datetime(3) DEFAULT NULL COMMENT 'Last registration time',
  `fs_instance_id` bigint unsigned DEFAULT NULL COMMENT 'Owning FS instance ID',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Remark',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_dn` (`tenant_id`,`dn_number`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_agent_id` (`agent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Extension table';

-- ============================================================
-- 8. ACD queue table
-- ============================================================
DROP TABLE IF EXISTS `tb_queue`;
CREATE TABLE `tb_queue` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Queue ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  `queue_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Queue name',
  `queue_code` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Queue code',
  `queue_type` bigint DEFAULT '1' COMMENT 'Queue type: 1-inbound 2-outbound',
  `strategy` varchar(30) COLLATE utf8mb4_unicode_ci DEFAULT 'round_robin' COMMENT 'Allocation strategy: round_robin-round robin longest_idle-longest idle skill_level-skill level',
  `max_wait_time` bigint DEFAULT '60' COMMENT 'Max wait time (seconds)',
  `max_wait_num` bigint DEFAULT '0' COMMENT 'Max waiting callers; 0-unlimited',
  `overflow_type` bigint DEFAULT '0' COMMENT 'Overflow handling: 0-hang up 1-transfer to queue 2-transfer to IVR 3-transfer to agent',
  `overflow_value` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Overflow value',
  `no_agent_type` bigint DEFAULT '0' COMMENT 'No-agent handling: 0-wait 1-overflow 2-hang up',
  `wrap_up_time` bigint DEFAULT '0' COMMENT 'After-call work time (seconds)',
  `skill_ids` text COLLATE utf8mb4_unicode_ci COMMENT 'Associated skill group ID list (JSON)',
  `status` bigint NOT NULL DEFAULT '1' COMMENT 'Status: 1-enabled 0-disabled',
  `wait_num` bigint DEFAULT '0' COMMENT 'Current waiting count',
  `agent_ready_num` bigint DEFAULT '0' COMMENT 'Ready agent count',
  `agent_busy_num` bigint DEFAULT '0' COMMENT 'Busy agent count',
  `sound_file` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Wait tone file',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Remark',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_queue_code` (`queue_code`),
  KEY `idx_tenant_id` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='ACD queue table';

-- ============================================================
-- 9. Inbound route table (access number DID -> queue/skill group)
-- ============================================================
DROP TABLE IF EXISTS `tb_inbound_route`;
CREATE TABLE `tb_inbound_route` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  `route_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Route name',
  `route_code` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Route code',
  `did` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Inbound access number (DID)',
  `queue_id` bigint unsigned DEFAULT NULL COMMENT 'Target queue ID',
  `queue_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Target queue name',
  `skill_group_id` bigint unsigned DEFAULT NULL COMMENT 'Target skill group ID (empty = unrestricted)',
  `welcome_sound` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Welcome tone file',
  `wait_sound` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Wait tone file',
  `work_time_type` bigint DEFAULT '0' COMMENT 'Work time type: 0-all day 1-custom',
  `work_time_config` text COLLATE utf8mb4_unicode_ci COMMENT 'Work time configuration (JSON)',
  `holiday_ids` text COLLATE utf8mb4_unicode_ci COMMENT 'Holiday ID list (JSON)',
  `off_work_type` bigint DEFAULT '0' COMMENT 'Off-work handling: 0-hang up 1-leave message 2-transfer to agent 3-transfer to queue',
  `off_work_value` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Off-work handling value',
  `off_work_sound` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Off-work prompt tone',
  `status` bigint NOT NULL DEFAULT '1' COMMENT 'Status: 1-enabled 0-disabled',
  `sort` bigint DEFAULT '0' COMMENT 'Sort order',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Remark',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_route_code` (`route_code`),
  UNIQUE KEY `uk_tenant_did` (`tenant_id`,`did`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_queue_id` (`queue_id`),
  KEY `idx_skill_group_id` (`skill_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Inbound route table';

-- ============================================================
-- 10. Line table (trunk physical channels)
-- ============================================================
DROP TABLE IF EXISTS `tb_line`;
CREATE TABLE `tb_line` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  `line_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Line name',
  `line_code` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Line code',
  `line_type` bigint DEFAULT '1' COMMENT 'Line type: 1-SIP trunk 2-PRI trunk 3-analog line',
  `provider` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Carrier',
  `gateway_ip` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Gateway address',
  `gateway_port` bigint DEFAULT '5060' COMMENT 'Gateway port',
  `username` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Username',
  `password` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Password',
  `caller_id` varchar(30) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Caller number',
  `max_channels` bigint DEFAULT '30' COMMENT 'Max concurrent channels',
  `used_channels` bigint DEFAULT '0' COMMENT 'Used concurrent channels',
  `status` bigint NOT NULL DEFAULT '1' COMMENT 'Status: 1-enabled 0-disabled',
  `register_status` bigint DEFAULT '0' COMMENT 'Registration status: 0-unregistered 1-registered',
  `skill_ids` text COLLATE utf8mb4_unicode_ci COMMENT 'Associated skill group ID list (JSON)',
  `fs_instance_id` bigint unsigned DEFAULT NULL COMMENT 'Owning FS instance ID',
  `last_register_at` datetime(3) DEFAULT NULL COMMENT 'Last registration time',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Remark',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_line_code` (`line_code`),
  KEY `idx_tenant_id` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Line table';

-- ============================================================
-- 11. Holiday schedule table
-- ============================================================
DROP TABLE IF EXISTS `tb_holiday_schedule`;
CREATE TABLE `tb_holiday_schedule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  `holiday_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Holiday name',
  `holiday_type` bigint DEFAULT '1' COMMENT 'Type: 1-statutory holiday 2-custom',
  `start_date` datetime(3) DEFAULT NULL COMMENT 'Start date',
  `end_date` datetime(3) DEFAULT NULL COMMENT 'End date',
  `time_config` text COLLATE utf8mb4_unicode_ci COMMENT 'Time configuration (JSON)',
  `status` bigint NOT NULL DEFAULT '1' COMMENT 'Status: 1-enabled 0-disabled',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Remark',
  PRIMARY KEY (`id`),
  KEY `idx_tenant_id` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Holiday schedule table';

-- ============================================================
-- 12. Call record table (CDR, RANGE partitioned by year)
--     Partitioned table constraint: every unique index must include the partition column start_time
-- ============================================================
DROP TABLE IF EXISTS `tb_call_record`;
CREATE TABLE `tb_call_record` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Record ID',
  `tenant_id` bigint unsigned NOT NULL COMMENT 'Tenant ID',
  `call_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Unique call ID',
  `call_type` bigint NOT NULL COMMENT 'Call type: 1-inbound 2-outbound 3-internal',
  `caller` varchar(30) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Caller number',
  `called` varchar(30) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Called number',
  `did` varchar(30) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Inbound access number (DID)',
  `agent_id` bigint unsigned DEFAULT NULL COMMENT 'Agent ID',
  `agent_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Agent number',
  `agent_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Agent name',
  `dn_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Extension number',
  `queue_id` bigint unsigned DEFAULT NULL COMMENT 'Queue ID',
  `queue_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Queue name',
  `skill_group_id` bigint unsigned DEFAULT NULL COMMENT 'Skill group ID',
  `skill_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Skill group name',
  `start_time` datetime(3) NOT NULL COMMENT 'Call start time',
  `answer_time` datetime(3) DEFAULT NULL COMMENT 'Answer time',
  `end_time` datetime(3) DEFAULT NULL COMMENT 'End time',
  `total_duration` bigint DEFAULT '0' COMMENT 'Total duration (seconds)',
  `talk_duration` bigint DEFAULT '0' COMMENT 'Talk duration (seconds)',
  `wait_duration` bigint DEFAULT '0' COMMENT 'Wait duration (seconds)',
  `wrap_up_duration` bigint DEFAULT '0' COMMENT 'After-call work duration (seconds)',
  `call_status` bigint DEFAULT '0' COMMENT 'Call status: 0-unanswered 1-answered 2-hung up 3-transferred',
  `answer_status` bigint DEFAULT '0' COMMENT 'Answer status: 0-unanswered 1-answered',
  `hangup_cause` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Hang-up cause',
  `hangup_direction` bigint DEFAULT '0' COMMENT 'Hang-up side: 0-unknown 1-caller 2-callee 3-system',
  `record_file` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Recording file path',
  `record_url` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Recording file URL',
  `record_duration` bigint DEFAULT '0' COMMENT 'Recording duration (seconds)',
  `transfer_count` bigint DEFAULT '0' COMMENT 'Transfer count',
  `hold_count` bigint DEFAULT '0' COMMENT 'Hold count',
  `hold_duration` bigint DEFAULT '0' COMMENT 'Hold duration (seconds)',
  `satisfaction` bigint DEFAULT '0' COMMENT 'Satisfaction score 0-5',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Remark',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  PRIMARY KEY (`id`,`start_time`),
  UNIQUE KEY `uk_call_id` (`call_id`,`start_time`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_caller` (`caller`),
  KEY `idx_called` (`called`),
  KEY `idx_agent_id` (`agent_id`),
  KEY `idx_start_time` (`start_time`),
  KEY `idx_call_type` (`call_type`),
  KEY `idx_did` (`did`),
  KEY `idx_queue_id` (`queue_id`),
  KEY `idx_end_time` (`end_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Call record table (CDR, partitioned by year)'
PARTITION BY RANGE (YEAR(`start_time`)) (
    PARTITION p2025 VALUES LESS THAN (2026),
    PARTITION p2026 VALUES LESS THAN (2027),
    PARTITION p2027 VALUES LESS THAN (2028),
    PARTITION p2028 VALUES LESS THAN (2029),
    PARTITION pmax VALUES LESS THAN MAXVALUE
);

-- ============================================================
-- 13. Agent service work-time record table (agentcenter runtime table)
--     Records the service work time for each agent check-in (duration per status); call details belong to tb_call_record
-- ============================================================
DROP TABLE IF EXISTS `tb_agent_service_record`;
CREATE TABLE `tb_agent_service_record` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Record ID',
  `tenant_id` bigint unsigned NOT NULL COMMENT 'Tenant ID',
  `record_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Unique record ID',
  `agent_id` bigint unsigned NOT NULL COMMENT 'Agent number',
  `agent_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Agent name',
  `call_id` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Associated call ID',
  `service_type` tinyint NOT NULL DEFAULT '1' COMMENT 'Service type: 1-inbound 2-outbound 3-internal',
  `checkin_time` datetime(3) DEFAULT NULL COMMENT 'Check-in time',
  `checkout_time` datetime(3) DEFAULT NULL COMMENT 'Check-out time',
  `talk_duration` int DEFAULT '0' COMMENT 'Talk duration (seconds)',
  `hold_duration` int DEFAULT '0' COMMENT 'Hold duration (seconds)',
  `ready_duration` int DEFAULT '0' COMMENT 'Ready duration (seconds)',
  `busy_duration` int DEFAULT '0' COMMENT 'Busy duration (seconds)',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_record_id` (`record_id`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_agent_id` (`agent_id`),
  KEY `idx_call_id` (`call_id`),
  KEY `idx_service_type` (`service_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent service work-time record table';

-- ============================================================
-- 14. Online agent table (agentcenter runtime table)
-- ============================================================
DROP TABLE IF EXISTS `tb_online_agent`;
CREATE TABLE `tb_online_agent` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned NOT NULL COMMENT 'Tenant ID',
  `agent_id` bigint unsigned NOT NULL COMMENT 'Agent ID',
  `agent_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Agent name',
  `agent_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Agent number',
  `dn_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Extension number',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT 'Agent status: 0-offline 1-checking in 2-checked in 3-ready 4-pre-allocated 5-ringing 6-outbound in progress 7-in call 8-after-call work 9-busy',
  `work_mode` tinyint DEFAULT '1' COMMENT 'Work mode: 1-inbound 2-outbound 3-inbound+outbound',
  `last_status` tinyint DEFAULT NULL COMMENT 'Previous status',
  `current_call_id` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Current call ID',
  `login_time` datetime(3) DEFAULT NULL COMMENT 'Login time',
  `checkin_time` datetime(3) DEFAULT NULL COMMENT 'Check-in time',
  `last_ready_time` datetime(3) DEFAULT NULL COMMENT 'Last ready time',
  `last_busy_time` datetime(3) DEFAULT NULL COMMENT 'Last busy time',
  `last_heartbeat` datetime(3) DEFAULT NULL COMMENT 'Last heartbeat time',
  `talk_count_today` int DEFAULT '0' COMMENT 'Today\'s call count',
  `talk_duration_today` int DEFAULT '0' COMMENT 'Today\'s talk duration (seconds)',
  `node_addr` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Node address',
  `ext_info` json DEFAULT NULL COMMENT 'Extended info',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_agent` (`tenant_id`,`agent_id`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_status` (`status`),
  KEY `idx_node_addr` (`node_addr`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Online agent table';

-- ============================================================
-- 15. FS instance registry table (FS node configs are managed manually through config files; this table is for registration only)
-- ============================================================
DROP TABLE IF EXISTS `tb_fs_instance`;
CREATE TABLE `tb_fs_instance` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `instance_code` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Instance code',
  `ip_address` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Host address',
  `esl_port` bigint DEFAULT '8021' COMMENT 'ESL port',
  `esl_password` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'ESL password',
  `sip_port` bigint DEFAULT '5060' COMMENT 'SIP port',
  `status` bigint DEFAULT '0' COMMENT 'Status: 0-offline 1-online 2-maintenance',
  `max_sessions` bigint DEFAULT '1000' COMMENT 'Max sessions',
  `current_session` bigint DEFAULT '0' COMMENT 'Current sessions',
  `cpu_usage` double DEFAULT '0' COMMENT 'CPU usage %',
  `mem_usage` double DEFAULT '0' COMMENT 'Memory usage %',
  `last_heartbeat` datetime(3) DEFAULT NULL COMMENT 'Last heartbeat time',
  `version` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Version',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Remark',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_instance_code` (`instance_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='FS instance registry table';

-- ============================================================
-- 16. Parameter configuration table
-- ============================================================
DROP TABLE IF EXISTS `tb_param_config`;
CREATE TABLE `tb_param_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID; 0 means global',
  `config_key` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Configuration key',
  `config_value` text COLLATE utf8mb4_unicode_ci COMMENT 'Configuration value',
  `config_type` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT 'string' COMMENT 'Configuration type: string/int/bool/json',
  `config_desc` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Configuration description',
  `is_system` bigint DEFAULT '0' COMMENT 'Is system configuration: 0-no 1-yes',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_config_key` (`tenant_id`,`config_key`),
  KEY `idx_tenant_id` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Parameter configuration table';

-- ============================================================
-- 17. Operation log table
-- ============================================================
DROP TABLE IF EXISTS `tb_operation_log`;
CREATE TABLE `tb_operation_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned DEFAULT NULL COMMENT 'Tenant ID',
  `operation_type` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Operation type: login/logout/add/update/delete/query',
  `module` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Operation module',
  `operation` varchar(200) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Operation content',
  `method` varchar(10) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Request method',
  `url` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Request URL',
  `request_data` text COLLATE utf8mb4_unicode_ci COMMENT 'Request parameters',
  `response_data` text COLLATE utf8mb4_unicode_ci COMMENT 'Response data',
  `user_id` bigint unsigned DEFAULT NULL COMMENT 'Operation user ID',
  `username` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Operation username',
  `ip` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Operation IP',
  `user_agent` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'User agent',
  `status` bigint DEFAULT '1' COMMENT 'Status: 1-success 0-failure',
  `error_msg` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Error message',
  `cost_time` bigint DEFAULT NULL COMMENT 'Duration (milliseconds)',
  `trace_id` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Trace ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  PRIMARY KEY (`id`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_trace_id` (`trace_id`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_module` (`module`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Operation log table';

-- ============================================================
-- 18. Form template table (dynamic forms)
-- ============================================================
DROP TABLE IF EXISTS `tb_form_template`;
CREATE TABLE `tb_form_template` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID',
  `template_key` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Template key (unique per tenant, e.g. customer)',
  `template_name` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Template name',
  `category` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'general' COMMENT 'Category code',
  `biz_type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'none' COMMENT 'Business type: none/customer/ticket',
  `current_version_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Current version ID',
  `status` bigint NOT NULL DEFAULT '1' COMMENT 'Status: 1-enabled 0-disabled',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Remark',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_template` (`tenant_id`,`template_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Form template table';

-- ============================================================
-- 18b. Form template version table (schema is immutable; each edit save adds a new version)
-- ============================================================
DROP TABLE IF EXISTS `tb_form_template_version`;
CREATE TABLE `tb_form_template_version` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID',
  `template_id` bigint unsigned NOT NULL COMMENT 'Template ID',
  `version` int NOT NULL COMMENT 'Version number (auto-increments within a template)',
  `schema` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Schema JSON for this version',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Publish time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_template_version` (`template_id`,`version`),
  KEY `idx_tenant` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Form template version table';

-- ============================================================
-- 18c. Customer master table (unique per tenant by phone number; aggregates multiple calls)
-- ============================================================
DROP TABLE IF EXISTS `tb_customer`;
CREATE TABLE `tb_customer` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID',
  `phone` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Customer phone number',
  `name` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Customer name',
  `data` mediumtext COLLATE utf8mb4_unicode_ci COMMENT 'Aggregated customer info JSON',
  `call_count` int NOT NULL DEFAULT '0' COMMENT 'Cumulative call/form-submission count',
  `first_seen_at` datetime(3) DEFAULT NULL COMMENT 'First seen',
  `last_seen_at` datetime(3) DEFAULT NULL COMMENT 'Last seen',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_phone` (`tenant_id`,`phone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Customer master table';

-- ============================================================
-- 19. Form record table (dynamic form data)
-- ============================================================
DROP TABLE IF EXISTS `tb_form_record`;
CREATE TABLE `tb_form_record` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Tenant ID',
  `template_id` bigint unsigned NOT NULL COMMENT 'Template ID',
  `template_version_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Bound template version ID',
  `template_key` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Template key (redundant, avoids join in queries)',
  `biz_type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'none' COMMENT 'Business type (copied from template at creation)',
  `biz_status` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'Ticket status: open/closed',
  `customer_id` bigint unsigned DEFAULT NULL COMMENT 'Associated customer master',
  `call_id` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Associated call ID',
  `direction` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Call direction: inbound/outbound',
  `agent_id` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Agent ID',
  `agent_name` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Agent name',
  `record_key` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Business key (e.g. customer phone number, nullable)',
  `data` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'Form data JSON: {"field name": value}',
  `field_snapshot` mediumtext COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'Flattened field snapshot at save time (name/label/type/options), used for list translation and export',
  `created_at` datetime(3) DEFAULT NULL COMMENT 'Creation time',
  `updated_at` datetime(3) DEFAULT NULL COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_record_key` (`tenant_id`,`template_key`,`record_key`),
  KEY `idx_tenant_template` (`tenant_id`,`template_key`),
  KEY `idx_biz_type` (`biz_type`),
  KEY `idx_customer_id` (`customer_id`),
  KEY `idx_call_id` (`call_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Form record table';

SET FOREIGN_KEY_CHECKS = 1;

-- ============================================================
-- Initialization data (required): minimal data so the system can start, log in, and be configured
-- ============================================================

-- Default tenant (referenced by tenant_id=1 across all business tables)
INSERT INTO `tb_tenant` (`tenant_name`, `tenant_code`, `status`, `max_agents`)
VALUES ('Default Tenant', 'default', 1, 1000);

-- System administrator account (password admin123, bcrypt hash; change the password after first login)
INSERT INTO `tb_agent`
    (`tenant_id`, `agent_number`, `agent_name`, `password`, `role`, `status`, `work_status`)
VALUES
    (1, 'admin', 'System Administrator',
     '$2a$10$JSTBWAqQUKdHRM62jsZmWugByinuayL8gPvUnol90OqmtSsQybEOC',
     'admin', 1, 0);