// Package service implements business logic.
// It invokes the repository layer for data operations.
package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strings"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
	"vocara/pkg/logger"
	"vocara/pkg/utils"

	"go.uber.org/zap"
)

// AgentService provides agent management operations.
type AgentService struct {
	agentRepo *repository.AgentRepository
	orgRepo   *repository.OrganizationRepository
	skillRepo *repository.SkillGroupRepository
	dnRepo    *repository.DNRepository
}

// NewAgentService creates an agent service.
func NewAgentService() *AgentService {
	return &AgentService{
		agentRepo: repository.NewAgentRepository(),
		orgRepo:   repository.NewOrganizationRepository(),
		skillRepo: repository.NewSkillGroupRepository(),
		dnRepo:    repository.NewDNRepository(),
	}
}

var (
	// mobilePattern matches an 11-digit mainland China mobile number starting with 1 and a second digit from 3 to 9.
	mobilePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)
	// emailPattern matches a standard email address.
	emailPattern = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)
)

// AgentSaveDTO contains an agent creation request.
type AgentSaveDTO struct {
	AgentNumber  string `json:"agent_number"`
	AgentName string `json:"agent_name"`
	Password  string `json:"password"`
	Mobile    string `json:"mobile"`
	Email     string `json:"email"`
	Gender    int    `json:"gender"`
	DN        string `json:"dn"`
	Role      string `json:"role"`
	Status    int    `json:"status"`
	OrgID     uint64 `json:"org_id"`
	Remark    string `json:"remark"`
}

// AgentUpdateDTO contains an agent update request.
type AgentUpdateDTO struct {
	ID        uint64 `json:"id"`
	AgentName string `json:"agent_name"`
	Mobile    string `json:"mobile"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	Gender    int    `json:"gender"`
	DN        string `json:"dn"`
	Role      string `json:"role"`
	Status    int    `json:"status"`
	OrgID     uint64 `json:"org_id"`
	Remark    string `json:"remark"`
}

// Page executes a paginated agent query.
func (s *AgentService) Page(ctx context.Context, tenantID uint64, q *repository.AgentPageQuery) ([]*model.Agent, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.agentRepo.PageQuery(ctx, tenantID, q)
}

// List returns agents matching a keyword.
func (s *AgentService) List(ctx context.Context, tenantID uint64, keyword string) ([]*model.Agent, error) {
	q := &repository.AgentPageQuery{
		Page:     1,
		PageSize: 1000,
		Keyword:  keyword,
	}
	list, _, err := s.agentRepo.PageQuery(ctx, tenantID, q)
	return list, err
}

// GetByID returns agent details by ID.
func (s *AgentService) GetByID(ctx context.Context, tenantID, id uint64) (*model.Agent, error) {
	return s.agentRepo.GetByID(ctx, tenantID, id)
}

// Create creates an agent.
func (s *AgentService) Create(ctx context.Context, tenantID uint64, dto *AgentSaveDTO) (*model.Agent, error) {
	// Validate parameters.
	if dto.AgentNumber == "" || dto.AgentName == "" {
		return nil, errors.ErrParamMissing
	}

	// Check whether the agent number exists.
	exists, err := s.agentRepo.ExistsByAgentNum(ctx, tenantID, dto.AgentNumber, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.ErrAlreadyExists.WithDetail("座席工号已存在")
	}

	// Use the same default password as ResetPassword and Import when none is provided.
	password := dto.Password
	if password == "" {
		password = "123456" // Default password.
	}

	// Hash the password.
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		logger.ErrorCtx(ctx, "hash password failed", zap.Error(err))
		return nil, errors.ErrSystemError
	}

	// Check that a specified extension exists and is available.
	if dto.DN != "" {
		dn, err := s.dnRepo.GetByDNNum(ctx, tenantID, dto.DN)
		if err != nil {
			// Continue without binding when the extension does not exist; return other errors.
			bizErr, ok := errors.IsBizError(err)
			if !ok || bizErr.Code != errors.CodeDNNotFound {
				return nil, err
			}
		}
		if dn != nil && dn.AgentID > 0 {
			return nil, errors.ErrDNNotAvailable.WithDetail("分机已被绑定")
		}
	}

	// Check that a specified organization exists.
	if dto.OrgID > 0 {
		org, err := s.orgRepo.GetByID(ctx, tenantID, dto.OrgID)
		if err != nil {
			return nil, err
		}
		_ = org
	}

	agent := &model.Agent{
		BaseModel: model.BaseModel{
			TenantID: tenantID,
		},
		AgentNumber:  dto.AgentNumber,
		AgentName: dto.AgentName,
		Password:  hashedPassword,
		Mobile:    dto.Mobile,
		Email:     dto.Email,
		Gender:    dto.Gender,
		DN:        dto.DN,
		Role:      dto.Role,
		Status:    dto.Status,
		OrgID:     dto.OrgID,
		Remark:    dto.Remark,
		WorkStatus: 0,
	}

	if agent.Role == "" {
		agent.Role = "agent"
	}
	if agent.Status == 0 {
		agent.Status = 1
	}

	err = s.agentRepo.Create(ctx, agent)
	if err != nil {
		return nil, err
	}

	// Update the extension binding when an extension is specified.
	if dto.DN != "" {
		dn, _ := s.dnRepo.GetByDNNum(ctx, tenantID, dto.DN)
		if dn != nil {
			_ = s.dnRepo.BindAgent(ctx, tenantID, dn.ID, agent.ID, agent.AgentNumber)
		}
	}

	return agent, nil
}

// Update updates an agent.
func (s *AgentService) Update(ctx context.Context, tenantID uint64, dto *AgentUpdateDTO) error {
	if dto.ID == 0 {
		return errors.ErrParamMissing
	}

	// Check whether the agent exists.
	agent, err := s.agentRepo.GetByID(ctx, tenantID, dto.ID)
	if err != nil {
		return err
	}

	// Update extension bindings when the extension changes.
	if dto.DN != agent.DN {
		// Unbind the current extension.
		if agent.DN != "" {
			oldDN, _ := s.dnRepo.GetByDNNum(ctx, tenantID, agent.DN)
			if oldDN != nil {
				_ = s.dnRepo.UnbindAgent(ctx, tenantID, oldDN.ID)
			}
		}
		// Bind the new extension.
		if dto.DN != "" {
			newDN, err := s.dnRepo.GetByDNNum(ctx, tenantID, dto.DN)
			if err != nil {
				return err
			}
			if newDN.AgentID > 0 && newDN.AgentID != dto.ID {
				return errors.ErrDNNotAvailable.WithDetail("分机已被绑定")
			}
			_ = s.dnRepo.BindAgent(ctx, tenantID, newDN.ID, dto.ID, agent.AgentNumber)
		}
	}

	agent.AgentName = dto.AgentName
	agent.Mobile = dto.Mobile
	agent.Email = dto.Email
	agent.Avatar = dto.Avatar
	agent.Gender = dto.Gender
	agent.DN = dto.DN
	agent.Role = dto.Role
	agent.Status = dto.Status
	agent.OrgID = dto.OrgID
	agent.Remark = dto.Remark

	return s.agentRepo.Update(ctx, agent)
}

// Delete deletes an agent.
func (s *AgentService) Delete(ctx context.Context, tenantID, id uint64) error {
	// Check whether the agent exists.
	agent, err := s.agentRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Unbind the extension before deleting the agent.
	if agent.DN != "" {
		dn, _ := s.dnRepo.GetByDNNum(ctx, tenantID, agent.DN)
		if dn != nil {
			_ = s.dnRepo.UnbindAgent(ctx, tenantID, dn.ID)
		}
	}

	// Delete the agent's skill group associations.
	_ = s.agentRepo.RemoveAgentSkills(ctx, tenantID, id)

	return s.agentRepo.Delete(ctx, tenantID, id)
}

// BatchDelete deletes agents in a batch.
func (s *AgentService) BatchDelete(ctx context.Context, tenantID uint64, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	return s.agentRepo.BatchDelete(ctx, tenantID, ids)
}

// UpdateStatus updates an agent's status.
func (s *AgentService) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return s.agentRepo.UpdateStatus(ctx, tenantID, id, status)
}

// UpdatePassword changes an agent's password.
func (s *AgentService) UpdatePassword(ctx context.Context, tenantID, id uint64, oldPassword, newPassword string) error {
	if oldPassword == "" {
		return errors.ErrParamMissing.WithDetail("旧密码不能为空")
	}
	if !utils.ValidatePasswordStrength(newPassword) {
		return errors.ErrParamInvalid.WithDetail("新密码长度为 8–20 位，须同时包含字母和数字")
	}

	agent, err := s.agentRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Verify the current password.
	if !utils.CheckPassword(oldPassword, agent.Password) {
		return errors.ErrAgentPasswordWrong
	}
	if oldPassword == newPassword {
		return errors.ErrParamInvalid.WithDetail("新密码不能与旧密码相同")
	}

	// Hash the new password.
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		logger.ErrorCtx(ctx, "hash password failed", zap.Error(err))
		return errors.ErrSystemError
	}

	return s.agentRepo.UpdatePassword(ctx, tenantID, id, hashedPassword)
}

// ProfileUpdateDTO contains the fields an agent may maintain in a self-service profile update.
type ProfileUpdateDTO struct {
	AgentName string `json:"agent_name"`
	Mobile    string `json:"mobile"`
	Email     string `json:"email"`
	Gender    int    `json:"gender"`
}

// UpdateProfile updates the current agent's profile using the identity from the JWT.
// The update whitelist excludes role, status, extension, organization, and other protected fields.
func (s *AgentService) UpdateProfile(ctx context.Context, tenantID, id uint64, dto *ProfileUpdateDTO) error {
	dto.AgentName = strings.TrimSpace(dto.AgentName)
	if dto.AgentName == "" {
		return errors.ErrParamMissing.WithDetail("姓名不能为空")
	}
	if len([]rune(dto.AgentName)) > 50 {
		return errors.ErrParamInvalid.WithDetail("姓名不能超过 50 个字符")
	}
	if dto.Mobile != "" && !mobilePattern.MatchString(dto.Mobile) {
		return errors.ErrParamInvalid.WithDetail("手机号格式不正确")
	}
	if dto.Email != "" && !emailPattern.MatchString(dto.Email) {
		return errors.ErrParamInvalid.WithDetail("邮箱格式不正确")
	}
	if dto.Gender < 0 || dto.Gender > 2 {
		return errors.ErrParamInvalid.WithDetail("性别取值非法")
	}

	fields := map[string]interface{}{
		"agent_name": dto.AgentName,
		"mobile":     dto.Mobile,
		"email":      dto.Email,
		"gender":     dto.Gender,
	}
	return s.agentRepo.UpdateProfile(ctx, tenantID, id, fields)
}

// ResetPassword resets an agent's password.
func (s *AgentService) ResetPassword(ctx context.Context, tenantID, id uint64, newPassword string) error {
	if newPassword == "" {
		newPassword = "123456" // Default password.
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		logger.ErrorCtx(ctx, "hash password failed", zap.Error(err))
		return errors.ErrSystemError
	}

	return s.agentRepo.UpdatePassword(ctx, tenantID, id, hashedPassword)
}

// AgentSkillAssignDTO contains an agent skill group assignment request.
type AgentSkillAssignDTO struct {
	AgentID    uint64            `json:"agent_id"`
	SkillItems []SkillAssignItem `json:"skill_items"`
}

type SkillAssignItem struct {
	SkillGroupID    uint64 `json:"skill_id"`
	SkillLevel int    `json:"skill_level"`
}

// AssignSkills assigns skill groups to an agent.
func (s *AgentService) AssignSkills(ctx context.Context, tenantID uint64, dto *AgentSkillAssignDTO) error {
	if dto.AgentID == 0 {
		return errors.ErrParamMissing
	}

	// Check whether the agent exists.
	agent, err := s.agentRepo.GetByID(ctx, tenantID, dto.AgentID)
	if err != nil {
		return err
	}

	// Replace the agent's current skill group associations.
	if err := s.agentRepo.RemoveAgentSkills(ctx, tenantID, dto.AgentID); err != nil {
		return err
	}

	// Add the requested skill group associations.
	if len(dto.SkillItems) > 0 {
		var skills []*model.AgentSkill
		for _, item := range dto.SkillItems {
			if item.SkillGroupID == 0 {
				continue
			}
			// Check whether the skill group exists.
			skill, err := s.skillRepo.GetByID(ctx, tenantID, item.SkillGroupID)
			if err != nil {
				continue
			}
			level := item.SkillLevel
			if level <= 0 {
				level = 5
			}
			skills = append(skills, &model.AgentSkill{
				TenantID:   tenantID,
				SkillGroupID:    skill.ID,
				AgentID:    agent.ID,
				AgentNumber:   agent.AgentNumber,
				AgentName:  agent.AgentName,
				SkillLevel: level,
			})
		}
		if len(skills) > 0 {
			if err := s.agentRepo.AddAgentSkills(ctx, skills); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetAgentSkills returns an agent's skill groups.
func (s *AgentService) GetAgentSkills(ctx context.Context, tenantID, agentID uint64) ([]*model.AgentSkill, error) {
	return s.agentRepo.GetAgentSkills(ctx, tenantID, agentID)
}


// Import batch imports agents from CSV.
func (s *AgentService) Import(ctx context.Context, tenantID uint64, reader io.Reader) (int, int, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	// Read the CSV header.
	headers, err := csvReader.Read()
	if err != nil {
		return 0, 0, errors.WrapWithCode(err, errors.CodeParamError, "CSV文件格式错误")
	}

	// Build the column index.
	colIndex := make(map[string]int)
	for i, h := range headers {
		colIndex[strings.TrimSpace(strings.ToLower(h))] = i
	}

	var successCount, failCount int
	var agents []*model.Agent
	rowNum := 0

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			failCount++
			continue
		}
		rowNum++

		agentNum := getCSVField(record, colIndex, "agent_number", "工号", "座席工号")
		agentName := getCSVField(record, colIndex, "agent_name", "姓名", "座席姓名")

		if agentNum == "" || agentName == "" {
			failCount++
			logger.WarnCtx(ctx, "import agent skip: missing required fields", zap.Int("row", rowNum))
			continue
		}

		// Check whether the agent number exists.
		exists, err := s.agentRepo.ExistsByAgentNum(ctx, tenantID, agentNum, 0)
		if err != nil {
			failCount++
			continue
		}
		if exists {
			failCount++
			continue
		}

		password := getCSVField(record, colIndex, "password", "密码")
		if password == "" {
			password = "123456"
		}
		hashedPassword, _ := utils.HashPassword(password)

		agent := &model.Agent{
			BaseModel: model.BaseModel{
				TenantID: tenantID,
			},
			AgentNumber:  agentNum,
			AgentName: agentName,
			Password:  hashedPassword,
			Mobile:    getCSVField(record, colIndex, "mobile", "手机号"),
			Email:     getCSVField(record, colIndex, "email", "邮箱"),
			DN:        getCSVField(record, colIndex, "dn", "分机号"),
			Role:      getCSVField(record, colIndex, "role", "角色"),
			Status:    1,
		}
		if agent.Role == "" {
			agent.Role = "agent"
		}

		agents = append(agents, agent)
		successCount++
	}

	if len(agents) > 0 {
		if err := s.agentRepo.BatchCreate(ctx, agents); err != nil {
			return 0, 0, err
		}
	}

	return successCount, failCount, nil
}

// Export exports agents as CSV.
func (s *AgentService) Export(ctx context.Context, tenantID uint64, q *repository.AgentPageQuery, writer io.Writer) error {
	list, err := s.agentRepo.ListExport(ctx, tenantID, q)
	if err != nil {
		return err
	}

	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write the CSV header.
	headers := []string{"座席工号", "座席姓名", "手机号", "邮箱", "分机号", "角色", "状态", "备注"}
	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	for _, agent := range list {
		role := agent.Role
		status := "禁用"
		if agent.Status == 1 {
			status = "启用"
		}
		row := []string{
			agent.AgentNumber,
			agent.AgentName,
			agent.Mobile,
			agent.Email,
			agent.DN,
			role,
			status,
			agent.Remark,
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// GetStatusCount returns agent counts by status.
func (s *AgentService) GetStatusCount(ctx context.Context, tenantID uint64) (map[string]int64, error) {
	return s.agentRepo.CountStatus(ctx, tenantID)
}

// Login authenticates an agent for admin login.
func (s *AgentService) Login(ctx context.Context, tenantID uint64, agentNum, password string) (*model.Agent, error) {
	agent, err := s.agentRepo.GetByAgentNum(ctx, tenantID, agentNum)
	if err != nil {
		return nil, errors.ErrAgentPasswordWrong
	}

	if agent.Status != 1 {
		return nil, errors.ErrForbidden.WithDetail("账号已被禁用")
	}

	if !utils.CheckPassword(password, agent.Password) {
		return nil, errors.ErrAgentPasswordWrong
	}

	return agent, nil
}

// ===== Helper Functions =====

func getCSVField(record []string, colIndex map[string]int, keys ...string) string {
	for _, key := range keys {
		if idx, ok := colIndex[strings.ToLower(key)]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
	}
	return ""
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

// FormatPageInfo calculates the number of pages.
func FormatPageInfo(page, pageSize int, total int64) (pages int) {
	if pageSize <= 0 {
		pageSize = 20
	}
	pages = int(total) / pageSize
	if int(total)%pageSize != 0 {
		pages++
	}
	return
}

// Keep fmt referenced.
var _ = fmt.Sprintf
