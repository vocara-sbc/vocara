package service

import (
	"context"

	"vocara/admin/internal/model"
)

// ticketSchema is the v1 JSON schema for the call ticket template. The regular expression \d is written as \\d in JSON.
const ticketSchema = `{"version":1,"formConfig":{"labelWidth":90,"labelPlacement":"left","size":"medium"},"fields":[{"id":"f_title","type":"input","name":"ticket_title","label":"工单标题","span":24,"props":{"placeholder":"简要概括问题","maxlength":100},"rules":{"required":true,"pattern":"","patternHint":"","maxLength":100},"defaultValue":"","showInTable":true,"tableWidth":160,"searchable":true,"sort":1},{"id":"f_phone","type":"input","name":"phone","label":"客户电话","span":12,"props":{"placeholder":"客户手机号","maxlength":20},"rules":{"required":true,"pattern":"^1[3-9]\\d{9}$","patternHint":"请输入正确的手机号","maxLength":20},"defaultValue":"","showInTable":true,"tableWidth":130,"searchable":true,"sort":2},{"id":"f_type","type":"select","name":"ticket_type","label":"工单类型","span":12,"props":{"options":[{"label":"咨询","value":"consult"},{"label":"投诉","value":"complaint"},{"label":"故障","value":"fault"},{"label":"其他","value":"other"}]},"rules":{},"defaultValue":null,"showInTable":true,"tableWidth":100,"searchable":false,"sort":3},{"id":"f_priority","type":"radio","name":"priority","label":"紧急程度","span":12,"props":{"options":[{"label":"普通","value":"normal"},{"label":"紧急","value":"urgent"}]},"rules":{},"defaultValue":"","showInTable":true,"tableWidth":100,"searchable":false,"sort":4},{"id":"f_content","type":"textarea","name":"content","label":"工单内容","span":24,"props":{"rows":4,"maxlength":1000,"placeholder":"详细描述问题"},"rules":{"required":true,"maxLength":1000},"defaultValue":"","showInTable":false,"tableWidth":0,"searchable":false,"sort":5}]}`

// customerSchema is the v1 JSON schema for the customer information template. The phone field is required and customer_name is optional.
const customerSchema = `{"version":1,"formConfig":{"labelWidth":90,"labelPlacement":"left","size":"medium"},"fields":[{"id":"f_phone","type":"input","name":"phone","label":"客户手机号","span":12,"props":{"placeholder":"客户手机号","maxlength":20},"rules":{"required":true,"pattern":"","patternHint":"","maxLength":20},"defaultValue":"","showInTable":true,"tableWidth":130,"searchable":true,"sort":1},{"id":"f_name","type":"input","name":"customer_name","label":"客户姓名","span":12,"props":{"placeholder":"客户姓名","maxlength":50},"rules":{"required":false,"pattern":"","patternHint":"","maxLength":50},"defaultValue":"","showInTable":true,"tableWidth":120,"searchable":true,"sort":2}]}`

// EnsureBuiltinTemplates performs idempotent initialization at startup by dropping the unused schema column and creating the built-in customer and ticket templates or correcting their biz_type when needed.
func (s *FormService) EnsureBuiltinTemplates(ctx context.Context) error {
	const tenantID uint64 = 1
	db := s.templateRepo.GetDB()

	// The unused schema NOT NULL column is not managed by the model, and AutoMigrate does not drop columns.
	// EnsureTemplateVersions populates v1 data before the column is dropped, preventing Error 1364 when inserting new templates.
	if db.Migrator().HasColumn(&model.FormTemplate{}, "schema") {
		if err := db.Migrator().DropColumn(&model.FormTemplate{}, "schema"); err != nil {
			return err
		}
	}

	// Create the customer template if it is absent, or correct its business type marker if it exists.
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

	// Skip creation when the ticket template already exists.
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
