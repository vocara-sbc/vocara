package service

import (
	"context"

	"vocara/admin/internal/model"
)

// ticketSchema 通话工单模板 v1 schema（JSON）。正则 \d 在 JSON 中写 \\d。
const ticketSchema = `{"version":1,"formConfig":{"labelWidth":90,"labelPlacement":"left","size":"medium"},"fields":[{"id":"f_title","type":"input","name":"ticket_title","label":"工单标题","span":24,"props":{"placeholder":"简要概括问题","maxlength":100},"rules":{"required":true,"pattern":"","patternHint":"","maxLength":100},"defaultValue":"","showInTable":true,"tableWidth":160,"searchable":true,"sort":1},{"id":"f_phone","type":"input","name":"phone","label":"客户电话","span":12,"props":{"placeholder":"客户手机号","maxlength":20},"rules":{"required":true,"pattern":"^1[3-9]\\d{9}$","patternHint":"请输入正确的手机号","maxLength":20},"defaultValue":"","showInTable":true,"tableWidth":130,"searchable":true,"sort":2},{"id":"f_type","type":"select","name":"ticket_type","label":"工单类型","span":12,"props":{"options":[{"label":"咨询","value":"consult"},{"label":"投诉","value":"complaint"},{"label":"故障","value":"fault"},{"label":"其他","value":"other"}]},"rules":{},"defaultValue":null,"showInTable":true,"tableWidth":100,"searchable":false,"sort":3},{"id":"f_priority","type":"radio","name":"priority","label":"紧急程度","span":12,"props":{"options":[{"label":"普通","value":"normal"},{"label":"紧急","value":"urgent"}]},"rules":{},"defaultValue":"","showInTable":true,"tableWidth":100,"searchable":false,"sort":4},{"id":"f_content","type":"textarea","name":"content","label":"工单内容","span":24,"props":{"rows":4,"maxlength":1000,"placeholder":"详细描述问题"},"rules":{"required":true,"maxLength":1000},"defaultValue":"","showInTable":false,"tableWidth":0,"searchable":false,"sort":5}]}`

// customerSchema 客户资料模板 v1 schema（JSON）。字段 phone 必填、customer_name 选填。
const customerSchema = `{"version":1,"formConfig":{"labelWidth":90,"labelPlacement":"left","size":"medium"},"fields":[{"id":"f_phone","type":"input","name":"phone","label":"客户手机号","span":12,"props":{"placeholder":"客户手机号","maxlength":20},"rules":{"required":true,"pattern":"","patternHint":"","maxLength":20},"defaultValue":"","showInTable":true,"tableWidth":130,"searchable":true,"sort":1},{"id":"f_name","type":"input","name":"customer_name","label":"客户姓名","span":12,"props":{"placeholder":"客户姓名","maxlength":50},"rules":{"required":false,"pattern":"","patternHint":"","maxLength":50},"defaultValue":"","showInTable":true,"tableWidth":120,"searchable":true,"sort":2}]}`

// EnsureBuiltinTemplates 启动幂等初始化：移除废弃 schema 列；customer / ticket 内置模板不存在则创建，存在则修正 biz_type
func (s *FormService) EnsureBuiltinTemplates(ctx context.Context) error {
	const tenantID uint64 = 1
	db := s.templateRepo.GetDB()

	// 旧表 schema NOT NULL 列在新模型中已废弃（AutoMigrate 不删列）；
	// 此时历史 v1 已由 EnsureTemplateVersions 补好，可安全删除，避免插入新模板时 Error 1364
	if db.Migrator().HasColumn(&model.FormTemplate{}, "schema") {
		if err := db.Migrator().DropColumn(&model.FormTemplate{}, "schema"); err != nil {
			return err
		}
	}

	// customer 模板：不存在则创建，存在则修正业务类型标记
	cust, err := s.templateRepo.GetByKey(ctx, tenantID, model.BizTypeCustomer)
	if err != nil {
		return err
	}
	if cust == nil {
		cust, err = s.TemplateCreate(ctx, tenantID, &FormTemplateSaveDTO{
			TemplateKey:  model.BizTypeCustomer,
			TemplateName: "客户资料",
			Category:     model.BizTypeCustomer,
			Schema:       customerSchema,
			Remark:       "客户资料模板",
		})
		if err != nil {
			return err
		}
		if err := s.templateRepo.SetBizType(ctx, cust.ID, model.BizTypeCustomer); err != nil {
			return err
		}
	} else if cust.BizType != model.BizTypeCustomer {
		if err := s.templateRepo.SetBizType(ctx, cust.ID, model.BizTypeCustomer); err != nil {
			return err
		}
	}

	// ticket 模板已存在则跳过
	tkt, err := s.templateRepo.GetByKey(ctx, tenantID, model.BizTypeTicket)
	if err != nil {
		return err
	}
	if tkt != nil {
		return nil
	}
	tpl, err := s.TemplateCreate(ctx, tenantID, &FormTemplateSaveDTO{
		TemplateKey:  model.BizTypeTicket,
		TemplateName: "通话工单",
		Category:     model.BizTypeTicket,
		Schema:       ticketSchema,
		Remark:       "通话工单模板",
	})
	if err != nil {
		return err
	}
	return s.templateRepo.SetBizType(ctx, tpl.ID, model.BizTypeTicket)
}
