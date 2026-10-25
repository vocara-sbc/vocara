-- ============================================================
-- Vocara 呼叫中心系统 - 数据库初始化脚本（基线）
-- Database: MySQL 8.0+
-- 命名规范：
--   1. 表名统一 tb_ 前缀 + 业务语义（tb_inbound_route、tb_line 等）；
--   2. 主键统一 id（bigint unsigned 自增）；主键引用统一 xxx_id（bigint unsigned）；
--   3. 工号/号码类字符串统一 xxx_number（agent_number/dn_number/did），编码类统一 xxx_code；
--   4. 时间字段统一 created_at / updated_at（datetime(3) DEFAULT NULL）；
--   5. 租户字段统一 tenant_id（bigint unsigned）；
--   6. 状态字段统一 status（1-启用 0-禁用），特殊状态机另行注释；
--   7. 字段与 GORM 模型（services/admin、services/agentcenter）保持一致。
-- ============================================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================================
-- 1. 租户表
-- ============================================================
DROP TABLE IF EXISTS `tb_tenant`;
CREATE TABLE `tb_tenant` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '租户ID',
  `tenant_name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '租户名称',
  `tenant_code` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '租户编码',
  `contact_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '联系人',
  `contact_phone` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '联系电话',
  `contact_email` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '联系邮箱',
  `address` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '地址',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '状态: 1-启用 0-禁用',
  `expire_time` datetime(3) DEFAULT NULL COMMENT '过期时间',
  `max_agents` bigint NOT NULL DEFAULT '0' COMMENT '最大座席数',
  `max_lines` bigint NOT NULL DEFAULT '0' COMMENT '最大线路数',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_code` (`tenant_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='租户表';

-- ============================================================
-- 2. 座席表（含管理后台登录账号，role=admin/supervisor 可登录管理后台）
-- ============================================================
DROP TABLE IF EXISTS `tb_agent`;
CREATE TABLE `tb_agent` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '座席ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `agent_number` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '座席工号（登录账号）',
  `agent_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '座席姓名',
  `password` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '登录密码(bcrypt 哈希)',
  `mobile` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '手机号',
  `email` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '邮箱',
  `avatar` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '头像URL',
  `gender` bigint DEFAULT '0' COMMENT '性别: 0-未知 1-男 2-女',
  `dn_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '绑定分机号',
  `role` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT 'agent' COMMENT '角色: admin-管理员 supervisor-班长 agent-座席',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '状态: 1-启用 0-禁用',
  `work_status` bigint NOT NULL DEFAULT '0' COMMENT '工作状态: 0-离线 1-空闲 2-忙碌 3-通话中 4-休息 5-话后处理',
  `group_id` bigint unsigned DEFAULT NULL COMMENT '所属座席组ID',
  `skill_levels` text COLLATE utf8mb4_unicode_ci COMMENT '技能等级(JSON)',
  `last_login_at` datetime(3) DEFAULT NULL COMMENT '最后登录时间',
  `last_login_ip` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '最后登录IP',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_agent_number` (`tenant_id`,`agent_number`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='座席表';

-- ============================================================
-- 3. 组织机构表
-- ============================================================
DROP TABLE IF EXISTS `tb_organization`;
CREATE TABLE `tb_organization` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '机构ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID',
  `org_name` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '机构/部门名称',
  `org_code` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '机构编码',
  `parent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '父机构ID（0为根）',
  `level` bigint NOT NULL DEFAULT '1' COMMENT '层级（根=1）',
  `path` varchar(1000) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '祖先ID路径，逗号分隔',
  `leader_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '负责人ID',
  `leader_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '负责人姓名',
  `sort` bigint NOT NULL DEFAULT '0' COMMENT '同级排序',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '状态: 1-启用 0-禁用',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_tenant_org` (`tenant_id`,`org_code`),
  KEY `idx_parent_id` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组织机构表';

-- ============================================================
-- 4. 技能组表
-- ============================================================
DROP TABLE IF EXISTS `tb_skill_group`;
CREATE TABLE `tb_skill_group` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '技能组ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `skill_code` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '技能组编码',
  `skill_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '技能组名称',
  `skill_type` bigint DEFAULT '1' COMMENT '技能类型: 1-呼入 2-呼出 3-混合型',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '状态: 1-启用 0-禁用',
  `priority` bigint DEFAULT '0' COMMENT '优先级',
  `agent_count` bigint DEFAULT '0' COMMENT '座席数量',
  `description` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '描述',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_skill_code` (`skill_code`),
  KEY `idx_tenant_id` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='技能组表';

-- ============================================================
-- 6. 座席-技能组关系表（admin 与 agentcenter 共用）
-- ============================================================
DROP TABLE IF EXISTS `tb_agent_skill`;
CREATE TABLE `tb_agent_skill` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned NOT NULL COMMENT '租户ID',
  `skill_group_id` bigint unsigned NOT NULL COMMENT '技能组ID',
  `agent_id` bigint unsigned NOT NULL COMMENT '座席ID',
  `agent_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '座席工号',
  `agent_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '座席姓名',
  `skill_level` bigint DEFAULT '5' COMMENT '技能等级 1-10',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_skill` (`agent_id`,`skill_group_id`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_skill_agent` (`skill_group_id`,`agent_id`),
  KEY `idx_agent_number` (`agent_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='座席-技能组关系表';

-- ============================================================
-- 7. 分机表（SIP 注册凭据，FS 动态目录数据源）
-- ============================================================
DROP TABLE IF EXISTS `tb_dn`;
CREATE TABLE `tb_dn` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '分机ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `dn_number` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '分机号码',
  `dn_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '分机名称',
  `password` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT 'SIP 注册密码',
  `dn_type` bigint DEFAULT '1' COMMENT '分机类型: 1-座席分机 2-通用分机',
  `agent_id` bigint unsigned DEFAULT NULL COMMENT '绑定座席ID',
  `agent_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '绑定座席工号',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '状态: 1-启用 0-禁用',
  `register_status` bigint DEFAULT '0' COMMENT '注册状态: 0-未注册 1-已注册',
  `ip_address` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '注册IP',
  `user_agent` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '注册UA',
  `last_register_at` datetime(3) DEFAULT NULL COMMENT '最后注册时间',
  `fs_instance_id` bigint unsigned DEFAULT NULL COMMENT '所属FS实例ID',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_dn` (`tenant_id`,`dn_number`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_agent_id` (`agent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='分机表';

-- ============================================================
-- 8. ACD 队列表
-- ============================================================
DROP TABLE IF EXISTS `tb_queue`;
CREATE TABLE `tb_queue` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '队列ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `queue_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '队列名称',
  `queue_code` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '队列编码',
  `queue_type` bigint DEFAULT '1' COMMENT '队列类型: 1-呼入 2-呼出',
  `strategy` varchar(30) COLLATE utf8mb4_unicode_ci DEFAULT 'round_robin' COMMENT '分配策略: round_robin-轮询 longest_idle-最长空闲 skill_level-技能等级',
  `max_wait_time` bigint DEFAULT '60' COMMENT '最大等待时间(秒)',
  `max_wait_num` bigint DEFAULT '0' COMMENT '最大等待人数 0-不限制',
  `overflow_type` bigint DEFAULT '0' COMMENT '溢出处理: 0-挂断 1-转队列 2-转IVR 3-转座席',
  `overflow_value` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '溢出值',
  `no_agent_type` bigint DEFAULT '0' COMMENT '无座席处理: 0-等待 1-溢出 2-挂断',
  `wrap_up_time` bigint DEFAULT '0' COMMENT '话后处理时间(秒)',
  `skill_ids` text COLLATE utf8mb4_unicode_ci COMMENT '关联技能组ID列表(JSON)',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '状态: 1-启用 0-禁用',
  `wait_num` bigint DEFAULT '0' COMMENT '当前等待数',
  `agent_ready_num` bigint DEFAULT '0' COMMENT '就绪座席数',
  `agent_busy_num` bigint DEFAULT '0' COMMENT '忙碌座席数',
  `sound_file` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '等待音文件',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_queue_code` (`queue_code`),
  KEY `idx_tenant_id` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='ACD队列表';

-- ============================================================
-- 9. 呼入路由表（接入号 DID → 队列/技能组）
-- ============================================================
DROP TABLE IF EXISTS `tb_inbound_route`;
CREATE TABLE `tb_inbound_route` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `route_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '路由名称',
  `route_code` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '路由编码',
  `did` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '呼入接入号(DID)',
  `queue_id` bigint unsigned DEFAULT NULL COMMENT '目标队列ID',
  `queue_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '目标队列名称',
  `skill_group_id` bigint unsigned DEFAULT NULL COMMENT '目标技能组ID（空为不限）',
  `welcome_sound` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '欢迎音文件',
  `wait_sound` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '等待音文件',
  `work_time_type` bigint DEFAULT '0' COMMENT '工作时间类型: 0-全天候 1-自定义',
  `work_time_config` text COLLATE utf8mb4_unicode_ci COMMENT '工作时间配置(JSON)',
  `holiday_ids` text COLLATE utf8mb4_unicode_ci COMMENT '节假日ID列表(JSON)',
  `off_work_type` bigint DEFAULT '0' COMMENT '非工作时间处理: 0-挂断 1-留言 2-转座席 3-转队列',
  `off_work_value` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '非工作时间处理值',
  `off_work_sound` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '非工作时间提示音',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '状态: 1-启用 0-禁用',
  `sort` bigint DEFAULT '0' COMMENT '排序',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_route_code` (`route_code`),
  UNIQUE KEY `uk_tenant_did` (`tenant_id`,`did`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_queue_id` (`queue_id`),
  KEY `idx_skill_group_id` (`skill_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='呼入路由表';

-- ============================================================
-- 10. 线路表（中继物理通道）
-- ============================================================
DROP TABLE IF EXISTS `tb_line`;
CREATE TABLE `tb_line` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `line_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '线路名称',
  `line_code` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '线路编码',
  `line_type` bigint DEFAULT '1' COMMENT '线路类型: 1-SIP中继 2-PRI中继 3-模拟线',
  `provider` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '运营商',
  `gateway_ip` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '网关地址',
  `gateway_port` bigint DEFAULT '5060' COMMENT '网关端口',
  `username` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '用户名',
  `password` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '密码',
  `caller_id` varchar(30) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '主叫号码',
  `max_channels` bigint DEFAULT '30' COMMENT '最大并发数',
  `used_channels` bigint DEFAULT '0' COMMENT '已用并发数',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '状态: 1-启用 0-禁用',
  `register_status` bigint DEFAULT '0' COMMENT '注册状态: 0-未注册 1-已注册',
  `skill_ids` text COLLATE utf8mb4_unicode_ci COMMENT '关联技能组ID列表(JSON)',
  `fs_instance_id` bigint unsigned DEFAULT NULL COMMENT '所属FS实例ID',
  `last_register_at` datetime(3) DEFAULT NULL COMMENT '最后注册时间',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_line_code` (`line_code`),
  KEY `idx_tenant_id` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='线路表';

-- ============================================================
-- 11. 节假日配置表
-- ============================================================
DROP TABLE IF EXISTS `tb_holiday_schedule`;
CREATE TABLE `tb_holiday_schedule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `holiday_name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '节假日名称',
  `holiday_type` bigint DEFAULT '1' COMMENT '类型: 1-法定假日 2-自定义',
  `start_date` datetime(3) DEFAULT NULL COMMENT '开始日期',
  `end_date` datetime(3) DEFAULT NULL COMMENT '结束日期',
  `time_config` text COLLATE utf8mb4_unicode_ci COMMENT '时间配置(JSON)',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '状态: 1-启用 0-禁用',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  KEY `idx_tenant_id` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节假日配置表';

-- ============================================================
-- 12. 通话记录表（CDR，按年 RANGE 分区）
--     分区表约束：所有唯一索引必须包含分区列 start_time
-- ============================================================
DROP TABLE IF EXISTS `tb_call_record`;
CREATE TABLE `tb_call_record` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '记录ID',
  `tenant_id` bigint unsigned NOT NULL COMMENT '租户ID',
  `call_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '通话唯一ID',
  `call_type` bigint NOT NULL COMMENT '通话类型: 1-呼入 2-呼出 3-内部呼叫',
  `caller` varchar(30) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '主叫号码',
  `called` varchar(30) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '被叫号码',
  `did` varchar(30) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '呼入接入号(DID)',
  `agent_id` bigint unsigned DEFAULT NULL COMMENT '座席ID',
  `agent_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '座席工号',
  `agent_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '座席姓名',
  `dn_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '分机号',
  `queue_id` bigint unsigned DEFAULT NULL COMMENT '队列ID',
  `queue_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '队列名称',
  `skill_group_id` bigint unsigned DEFAULT NULL COMMENT '技能组ID',
  `skill_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '技能组名称',
  `start_time` datetime(3) NOT NULL COMMENT '通话开始时间',
  `answer_time` datetime(3) DEFAULT NULL COMMENT '接听时间',
  `end_time` datetime(3) DEFAULT NULL COMMENT '结束时间',
  `total_duration` bigint DEFAULT '0' COMMENT '总时长(秒)',
  `talk_duration` bigint DEFAULT '0' COMMENT '通话时长(秒)',
  `wait_duration` bigint DEFAULT '0' COMMENT '等待时长(秒)',
  `wrap_up_duration` bigint DEFAULT '0' COMMENT '话后处理时长(秒)',
  `call_status` bigint DEFAULT '0' COMMENT '通话状态: 0-未接 1-已接 2-已挂断 3-转接',
  `answer_status` bigint DEFAULT '0' COMMENT '接听状态: 0-未接 1-已接',
  `hangup_cause` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '挂机原因',
  `hangup_direction` bigint DEFAULT '0' COMMENT '挂断方: 0-未知 1-主叫 2-被叫 3-系统',
  `record_file` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '录音文件路径',
  `record_url` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '录音文件URL',
  `record_duration` bigint DEFAULT '0' COMMENT '录音时长(秒)',
  `transfer_count` bigint DEFAULT '0' COMMENT '转接次数',
  `hold_count` bigint DEFAULT '0' COMMENT '保持次数',
  `hold_duration` bigint DEFAULT '0' COMMENT '保持时长(秒)',
  `satisfaction` bigint DEFAULT '0' COMMENT '满意度评分 0-5',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通话记录表(CDR，按年分区)'
PARTITION BY RANGE (YEAR(`start_time`)) (
    PARTITION p2025 VALUES LESS THAN (2026),
    PARTITION p2026 VALUES LESS THAN (2027),
    PARTITION p2027 VALUES LESS THAN (2028),
    PARTITION p2028 VALUES LESS THAN (2029),
    PARTITION pmax VALUES LESS THAN MAXVALUE
);

-- ============================================================
-- 13. 座席服务工时记录表（agentcenter 运行时表）
--     记录座席每次签入的服务工时（各状态时长），通话明细归 tb_call_record
-- ============================================================
DROP TABLE IF EXISTS `tb_agent_service_record`;
CREATE TABLE `tb_agent_service_record` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '记录ID',
  `tenant_id` bigint unsigned NOT NULL COMMENT '租户ID',
  `record_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '记录唯一ID',
  `agent_id` bigint unsigned NOT NULL COMMENT '座席工号',
  `agent_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '座席姓名',
  `call_id` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '关联通话ID',
  `service_type` tinyint NOT NULL DEFAULT '1' COMMENT '服务类型: 1-呼入 2-外呼 3-内部呼叫',
  `checkin_time` datetime(3) DEFAULT NULL COMMENT '签入时间',
  `checkout_time` datetime(3) DEFAULT NULL COMMENT '签出时间',
  `talk_duration` int DEFAULT '0' COMMENT '通话时长(秒)',
  `hold_duration` int DEFAULT '0' COMMENT '保持时长(秒)',
  `ready_duration` int DEFAULT '0' COMMENT '就绪时长(秒)',
  `busy_duration` int DEFAULT '0' COMMENT '示忙时长(秒)',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_record_id` (`record_id`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_agent_id` (`agent_id`),
  KEY `idx_call_id` (`call_id`),
  KEY `idx_service_type` (`service_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='座席服务工时记录表';

-- ============================================================
-- 14. 在线座席表（agentcenter 运行时表）
-- ============================================================
DROP TABLE IF EXISTS `tb_online_agent`;
CREATE TABLE `tb_online_agent` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned NOT NULL COMMENT '租户ID',
  `agent_id` bigint unsigned NOT NULL COMMENT '座席ID',
  `agent_name` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '座席姓名',
  `agent_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '座席工号',
  `dn_number` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '分机号',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '座席状态: 0-离线 1-签入中 2-已签入 3-示闲 4-预分配 5-振铃 6-外呼中 7-通话中 8-话后处理 9-示忙',
  `work_mode` tinyint DEFAULT '1' COMMENT '工作模式: 1-呼入 2-呼出 3-呼入+呼出',
  `last_status` tinyint DEFAULT NULL COMMENT '上一个状态',
  `current_call_id` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '当前通话ID',
  `login_time` datetime(3) DEFAULT NULL COMMENT '登录时间',
  `checkin_time` datetime(3) DEFAULT NULL COMMENT '签入时间',
  `last_ready_time` datetime(3) DEFAULT NULL COMMENT '最后示闲时间',
  `last_busy_time` datetime(3) DEFAULT NULL COMMENT '最后示忙时间',
  `last_heartbeat` datetime(3) DEFAULT NULL COMMENT '最后心跳时间',
  `talk_count_today` int DEFAULT '0' COMMENT '今日通话数',
  `talk_duration_today` int DEFAULT '0' COMMENT '今日通话时长(秒)',
  `node_addr` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '所在节点地址',
  `ext_info` json DEFAULT NULL COMMENT '扩展信息',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_agent` (`tenant_id`,`agent_id`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_status` (`status`),
  KEY `idx_node_addr` (`node_addr`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='在线座席表';

-- ============================================================
-- 15. FS 实例登记表（FS 节点配置以配置文件手工管理，本表仅作登记）
-- ============================================================
DROP TABLE IF EXISTS `tb_fs_instance`;
CREATE TABLE `tb_fs_instance` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `instance_code` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '实例编码',
  `ip_address` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '主机地址',
  `esl_port` bigint DEFAULT '8021' COMMENT 'ESL端口',
  `esl_password` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT 'ESL密码',
  `sip_port` bigint DEFAULT '5060' COMMENT 'SIP端口',
  `status` bigint DEFAULT '0' COMMENT '状态: 0-离线 1-在线 2-维护中',
  `max_sessions` bigint DEFAULT '1000' COMMENT '最大会话数',
  `current_session` bigint DEFAULT '0' COMMENT '当前会话数',
  `cpu_usage` double DEFAULT '0' COMMENT 'CPU使用率%',
  `mem_usage` double DEFAULT '0' COMMENT '内存使用率%',
  `last_heartbeat` datetime(3) DEFAULT NULL COMMENT '最后心跳时间',
  `version` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '版本号',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_instance_code` (`instance_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='FS实例登记表';

-- ============================================================
-- 16. 参数配置表
-- ============================================================
DROP TABLE IF EXISTS `tb_param_config`;
CREATE TABLE `tb_param_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID 0表示全局',
  `config_key` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '配置键',
  `config_value` text COLLATE utf8mb4_unicode_ci COMMENT '配置值',
  `config_type` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT 'string' COMMENT '配置类型: string/int/bool/json',
  `config_desc` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '配置描述',
  `is_system` bigint DEFAULT '0' COMMENT '是否系统配置: 0-否 1-是',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_config_key` (`tenant_id`,`config_key`),
  KEY `idx_tenant_id` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='参数配置表';

-- ============================================================
-- 17. 操作日志表
-- ============================================================
DROP TABLE IF EXISTS `tb_operation_log`;
CREATE TABLE `tb_operation_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tenant_id` bigint unsigned DEFAULT NULL COMMENT '租户ID',
  `operation_type` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '操作类型: login/logout/add/update/delete/query',
  `module` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '操作模块',
  `operation` varchar(200) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '操作内容',
  `method` varchar(10) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '请求方法',
  `url` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '请求URL',
  `request_data` text COLLATE utf8mb4_unicode_ci COMMENT '请求参数',
  `response_data` text COLLATE utf8mb4_unicode_ci COMMENT '响应数据',
  `user_id` bigint unsigned DEFAULT NULL COMMENT '操作用户ID',
  `username` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '操作用户名',
  `ip` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '操作IP',
  `user_agent` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '用户代理',
  `status` bigint DEFAULT '1' COMMENT '状态: 1-成功 0-失败',
  `error_msg` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '错误信息',
  `cost_time` bigint DEFAULT NULL COMMENT '耗时(毫秒)',
  `trace_id` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '追踪ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_tenant_id` (`tenant_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_trace_id` (`trace_id`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_module` (`module`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='操作日志表';

-- ============================================================
-- 18. 表单模板表（动态表单）
-- ============================================================
DROP TABLE IF EXISTS `tb_form_template`;
CREATE TABLE `tb_form_template` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID',
  `template_key` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '模板标识（租户内唯一，如 customer）',
  `template_name` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '模板名称',
  `category` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'general' COMMENT '分类编码',
  `biz_type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'none' COMMENT '业务类型: none/customer/ticket',
  `current_version_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '当前版本ID',
  `status` bigint NOT NULL DEFAULT '1' COMMENT '状态: 1-启用 0-禁用',
  `remark` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '备注',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_template` (`tenant_id`,`template_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='表单模板表';

-- ============================================================
-- 18b. 表单模板版本表（schema 不可变，每次编辑保存新增版本）
-- ============================================================
DROP TABLE IF EXISTS `tb_form_template_version`;
CREATE TABLE `tb_form_template_version` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID',
  `template_id` bigint unsigned NOT NULL COMMENT '模板ID',
  `version` int NOT NULL COMMENT '版本号（模板内自增）',
  `schema` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '该版本 schema JSON',
  `created_at` datetime(3) DEFAULT NULL COMMENT '发布时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_template_version` (`template_id`,`version`),
  KEY `idx_tenant` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='表单模板版本表';

-- ============================================================
-- 18c. 客户主档表（按手机号租户内唯一，多次来电聚合）
-- ============================================================
DROP TABLE IF EXISTS `tb_customer`;
CREATE TABLE `tb_customer` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID',
  `phone` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '客户手机号',
  `name` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '客户姓名',
  `data` mediumtext COLLATE utf8mb4_unicode_ci COMMENT '客户信息聚合 JSON',
  `call_count` int NOT NULL DEFAULT '0' COMMENT '累计来电/填报次数',
  `first_seen_at` datetime(3) DEFAULT NULL COMMENT '首次出现',
  `last_seen_at` datetime(3) DEFAULT NULL COMMENT '最近出现',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_phone` (`tenant_id`,`phone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客户主档表';

-- ============================================================
-- 19. 表单实例表（动态表单数据）
-- ============================================================
DROP TABLE IF EXISTS `tb_form_record`;
CREATE TABLE `tb_form_record` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `tenant_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '租户ID',
  `template_id` bigint unsigned NOT NULL COMMENT '模板ID',
  `template_version_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '绑定的模板版本ID',
  `template_key` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '模板标识（冗余，查询免 join）',
  `biz_type` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'none' COMMENT '业务类型（创建时从模板复制）',
  `biz_status` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '工单状态 open/closed',
  `customer_id` bigint unsigned DEFAULT NULL COMMENT '关联客户主档',
  `call_id` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '关联通话ID',
  `direction` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '呼叫方向 inbound/outbound',
  `agent_id` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '座席ID',
  `agent_name` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '座席名称',
  `record_key` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '业务主键（如客户手机号，可空）',
  `data` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '表单数据 JSON：{"字段name": 值}',
  `field_snapshot` mediumtext COLLATE utf8mb4_unicode_ci DEFAULT NULL COMMENT '保存时的拍平字段快照（name/label/type/options），列表翻译与导出用',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_record_key` (`tenant_id`,`template_key`,`record_key`),
  KEY `idx_tenant_template` (`tenant_id`,`template_key`),
  KEY `idx_biz_type` (`biz_type`),
  KEY `idx_customer_id` (`customer_id`),
  KEY `idx_call_id` (`call_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='表单实例表';

SET FOREIGN_KEY_CHECKS = 1;

-- ============================================================
-- 初始化数据（必需）：系统可启动、可登录、可配置的最低数据
-- ============================================================

-- 默认租户（全部业务表 tenant_id=1 引用）
INSERT INTO `tb_tenant` (`tenant_name`, `tenant_code`, `status`, `max_agents`)
VALUES ('默认租户', 'default', 1, 1000);

-- 系统管理员账号（密码 admin123，bcrypt 哈希，首次登录后请修改密码）
INSERT INTO `tb_agent`
    (`tenant_id`, `agent_number`, `agent_name`, `password`, `role`, `status`, `work_status`)
VALUES
    (1, 'admin', '系统管理员',
     '$2a$10$JSTBWAqQUKdHRM62jsZmWugByinuayL8gPvUnol90OqmtSsQybEOC',
     'admin', 1, 0);
