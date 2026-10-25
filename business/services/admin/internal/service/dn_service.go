package service

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
)

// DNService 分机服务
type DNService struct {
	dnRepo    *repository.DNRepository
	agentRepo *repository.AgentRepository
}

// NewDNService 创建分机服务
func NewDNService() *DNService {
	return &DNService{
		dnRepo:    repository.NewDNRepository(),
		agentRepo: repository.NewAgentRepository(),
	}
}

// DNSaveDTO 分机保存
type DNSaveDTO struct {
	DNNum        string `json:"dn_num"`
	DNName       string `json:"dn_name"`
	Password     string `json:"password"`
	DNType       int    `json:"dn_type"`
	Status       int    `json:"status"`
	FSInstanceID uint64 `json:"fs_instance_id"`
	Remark       string `json:"remark"`
}

// DNUpdateDTO 分机更新
type DNUpdateDTO struct {
	ID           uint64 `json:"id"`
	DNName       string `json:"dn_name"`
	Password     string `json:"password"`
	DNType       int    `json:"dn_type"`
	Status       int    `json:"status"`
	FSInstanceID uint64 `json:"fs_instance_id"`
	Remark       string `json:"remark"`
}

// DNBatchCreateDTO 批量创建分机
type DNBatchCreateDTO struct {
	StartDN      string `json:"start_dn"`
	Count        int    `json:"count"`
	Password     string `json:"password"`
	DNType       int    `json:"dn_type"`
	FSInstanceID uint64 `json:"fs_instance_id"`
	Prefix       string `json:"prefix"`
}

// Page 分页查询
func (s *DNService) Page(ctx context.Context, tenantID uint64, q *repository.DNPageQuery) ([]*model.DN, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.dnRepo.PageQuery(ctx, tenantID, q)
}

// List 查询列表
func (s *DNService) List(ctx context.Context, tenantID uint64) ([]*model.DN, error) {
	return s.dnRepo.ListAll(ctx, tenantID)
}

// GetByID 获取详情
func (s *DNService) GetByID(ctx context.Context, tenantID, id uint64) (*model.DN, error) {
	return s.dnRepo.GetByID(ctx, tenantID, id)
}

// Create 创建分机
func (s *DNService) Create(ctx context.Context, tenantID uint64, dto *DNSaveDTO) (*model.DN, error) {
	if dto.DNNum == "" {
		return nil, errors.ErrParamMissing.WithDetail("分机号不能为空")
	}

	exists, err := s.dnRepo.ExistsByDNNum(ctx, tenantID, dto.DNNum, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.ErrAlreadyExists.WithDetail("分机号已存在")
	}

	password := dto.Password
	if password == "" {
		password = "1234" // 默认 SIP 注册密码（与 FS default_password 一致，动态目录统一凭据源）
	}
	// 注册密码需存明文：FS 动态目录以 MD5(user:realm:password) 做 digest 鉴权，无法用 bcrypt 哈希（不同于 tb_agent 登录密码）。

	dn := &model.DN{
		BaseModel: model.BaseModel{
			TenantID: tenantID,
		},
		DNNum:        dto.DNNum,
		DNName:       dto.DNName,
		Password:     password,
		DNType:       dto.DNType,
		Status:       dto.Status,
		FSInstanceID: dto.FSInstanceID,
		Remark:       dto.Remark,
	}
	if dn.DNType == 0 {
		dn.DNType = 1
	}
	if dn.Status == 0 {
		dn.Status = 1
	}

	err = s.dnRepo.Create(ctx, dn)
	if err != nil {
		return nil, err
	}

	return dn, nil
}

// Update 更新
func (s *DNService) Update(ctx context.Context, tenantID uint64, dto *DNUpdateDTO) error {
	if dto.ID == 0 {
		return errors.ErrParamMissing
	}

	dn, err := s.dnRepo.GetByID(ctx, tenantID, dto.ID)
	if err != nil {
		return err
	}

	dn.DNName = dto.DNName
	dn.DNType = dto.DNType
	dn.Status = dto.Status
	dn.FSInstanceID = dto.FSInstanceID
	dn.Remark = dto.Remark

	if dto.Password != "" {
		// 注册密码存明文，供 FS 动态目录 digest 鉴权
		dn.Password = dto.Password
	}

	return s.dnRepo.Update(ctx, dn)
}

// Delete 删除
func (s *DNService) Delete(ctx context.Context, tenantID, id uint64) error {
	dn, err := s.dnRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// 如果绑定了座席，先解绑
	if dn.AgentID > 0 {
		_ = s.agentRepo.UpdateDNByID(ctx, tenantID, dn.AgentID, "")
	}

	return s.dnRepo.Delete(ctx, tenantID, id)
}

// UpdateStatus 更新状态
func (s *DNService) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return s.dnRepo.UpdateStatus(ctx, tenantID, id, status)
}

// BindAgent 绑定座席
func (s *DNService) BindAgent(ctx context.Context, tenantID, id, agentID uint64) error {
	agent, err := s.agentRepo.GetByID(ctx, tenantID, agentID)
	if err != nil {
		return err
	}

	dn, err := s.dnRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if dn.AgentID > 0 && dn.AgentID != agentID {
		return errors.ErrDNNotAvailable.WithDetail("分机已被绑定")
	}

	// 座席已绑定其他分机时先解绑
	if agent.DN != "" && agent.DN != dn.DNNum {
		oldDN, _ := s.dnRepo.GetByDNNum(ctx, tenantID, agent.DN)
		if oldDN != nil {
			_ = s.dnRepo.UnbindAgent(ctx, tenantID, oldDN.ID)
		}
	}

	err = s.dnRepo.BindAgent(ctx, tenantID, id, agentID, agent.AgentNumber)
	if err != nil {
		return err
	}

	// 更新座席的分机号
	return s.agentRepo.UpdateDNByID(ctx, tenantID, agentID, dn.DNNum)
}

// UnbindAgent 解绑座席
func (s *DNService) UnbindAgent(ctx context.Context, tenantID, id uint64) error {
	dn, err := s.dnRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if dn.AgentID > 0 {
		_ = s.agentRepo.UpdateDNByID(ctx, tenantID, dn.AgentID, "")
	}

	return s.dnRepo.UnbindAgent(ctx, tenantID, id)
}

// BatchCreate 批量创建分机
func (s *DNService) BatchCreate(ctx context.Context, tenantID uint64, dto *DNBatchCreateDTO) (int, error) {
	if dto.Count <= 0 || dto.Count > 1000 {
		return 0, errors.ErrParamInvalid.WithDetail("批量创建数量必须在1-1000之间")
	}

	password := dto.Password
	if password == "" {
		password = "1234" // 默认 SIP 注册密码（与 FS default_password 一致，动态目录统一凭据源）
	}

	var dns []*model.DN
	startNum := parseUint(dto.StartDN)
	if startNum == 0 {
		startNum = 1000
	}

	prefix := dto.Prefix
	dnType := dto.DNType
	if dnType == 0 {
		dnType = 1
	}

	count := 0
	for i := 0; i < dto.Count; i++ {
		dnNum := prefix + uint64ToStr(uint64(startNum+i))

		exists, err := s.dnRepo.ExistsByDNNum(ctx, tenantID, dnNum, 0)
		if err != nil {
			continue
		}
		if exists {
			continue
		}

		dns = append(dns, &model.DN{
			BaseModel: model.BaseModel{
				TenantID: tenantID,
			},
			DNNum:        dnNum,
			DNName:       "分机" + dnNum,
			Password:     password,
			DNType:       dnType,
			Status:       1,
			FSInstanceID: dto.FSInstanceID,
		})
		count++
	}

	if len(dns) > 0 {
		if err := s.dnRepo.BatchCreate(ctx, dns); err != nil {
			return 0, err
		}
	}

	return count, nil
}

// ListAvailable 查询可用分机
func (s *DNService) ListAvailable(ctx context.Context, tenantID uint64) ([]*model.DN, error) {
	return s.dnRepo.ListAvailable(ctx, tenantID)
}

func parseUint(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
