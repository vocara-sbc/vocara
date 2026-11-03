package service

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
)

// QueueService provides queue management services.
type QueueService struct {
	queueRepo *repository.QueueRepository
}

// NewQueueService creates a queue service.
func NewQueueService() *QueueService {
	return &QueueService{
		queueRepo: repository.NewQueueRepository(),
	}
}

// QueueSaveDTO contains queue data for creation.
type QueueSaveDTO struct {
	QueueName    string `json:"queue_name"`
	QueueCode    string `json:"queue_code"`
	QueueType    int    `json:"queue_type"`
	Strategy     string `json:"strategy"`
	MaxWaitTime  int    `json:"max_wait_time"`
	MaxWaitNum   int    `json:"max_wait_num"`
	OverflowType int    `json:"overflow_type"`
	OverflowValue string `json:"overflow_value"`
	NoAgentType  int    `json:"no_agent_type"`
	WrapUpTime   int    `json:"wrap_up_time"`
	SkillIDs     string `json:"skill_ids"`
	Status       int    `json:"status"`
	SoundFile    string `json:"sound_file"`
	Remark       string `json:"remark"`
}

// QueueUpdateDTO contains queue data for updates.
type QueueUpdateDTO struct {
	ID            uint64 `json:"id"`
	QueueName     string `json:"queue_name"`
	QueueType     int    `json:"queue_type"`
	Strategy      string `json:"strategy"`
	MaxWaitTime   int    `json:"max_wait_time"`
	MaxWaitNum    int    `json:"max_wait_num"`
	OverflowType  int    `json:"overflow_type"`
	OverflowValue string `json:"overflow_value"`
	NoAgentType   int    `json:"no_agent_type"`
	WrapUpTime    int    `json:"wrap_up_time"`
	SkillIDs      string `json:"skill_ids"`
	Status        int    `json:"status"`
	SoundFile     string `json:"sound_file"`
	Remark        string `json:"remark"`
}

// Page returns a paginated list of queues.
func (s *QueueService) Page(ctx context.Context, tenantID uint64, q *repository.QueuePageQuery) ([]*model.Queue, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.queueRepo.PageQuery(ctx, tenantID, q)
}

// List returns all queues.
func (s *QueueService) List(ctx context.Context, tenantID uint64) ([]*model.Queue, error) {
	return s.queueRepo.ListAll(ctx, tenantID)
}

// GetByID returns queue details.
func (s *QueueService) GetByID(ctx context.Context, tenantID, id uint64) (*model.Queue, error) {
	return s.queueRepo.GetByID(ctx, tenantID, id)
}

// Create creates a queue.
func (s *QueueService) Create(ctx context.Context, tenantID uint64, dto *QueueSaveDTO) (*model.Queue, error) {
	if dto.QueueName == "" || dto.QueueCode == "" {
		return nil, errors.ErrParamMissing
	}

	exists, err := s.queueRepo.ExistsByCode(ctx, tenantID, dto.QueueCode, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.ErrAlreadyExists.WithDetail("队列编码已存在")
	}

	queue := &model.Queue{
		BaseModel: model.BaseModel{
			TenantID: tenantID,
		},
		QueueName:     dto.QueueName,
		QueueCode:     dto.QueueCode,
		QueueType:     dto.QueueType,
		Strategy:      dto.Strategy,
		MaxWaitTime:   dto.MaxWaitTime,
		MaxWaitNum:    dto.MaxWaitNum,
		OverflowType:  dto.OverflowType,
		OverflowValue: dto.OverflowValue,
		NoAgentType:   dto.NoAgentType,
		WrapUpTime:    dto.WrapUpTime,
		SkillIDs:      dto.SkillIDs,
		Status:        dto.Status,
		SoundFile:     dto.SoundFile,
		Remark:        dto.Remark,
	}
	if queue.QueueType == 0 {
		queue.QueueType = 1
	}
	if queue.Strategy == "" {
		queue.Strategy = "round_robin"
	}
	if queue.MaxWaitTime == 0 {
		queue.MaxWaitTime = 60
	}
	if queue.Status == 0 {
		queue.Status = 1
	}

	err = s.queueRepo.Create(ctx, queue)
	if err != nil {
		return nil, err
	}

	return queue, nil
}

// Update updates a queue.
func (s *QueueService) Update(ctx context.Context, tenantID uint64, dto *QueueUpdateDTO) error {
	if dto.ID == 0 {
		return errors.ErrParamMissing
	}

	queue, err := s.queueRepo.GetByID(ctx, tenantID, dto.ID)
	if err != nil {
		return err
	}

	queue.QueueName = dto.QueueName
	queue.QueueType = dto.QueueType
	queue.Strategy = dto.Strategy
	queue.MaxWaitTime = dto.MaxWaitTime
	queue.MaxWaitNum = dto.MaxWaitNum
	queue.OverflowType = dto.OverflowType
	queue.OverflowValue = dto.OverflowValue
	queue.NoAgentType = dto.NoAgentType
	queue.WrapUpTime = dto.WrapUpTime
	queue.SkillIDs = dto.SkillIDs
	queue.Status = dto.Status
	queue.SoundFile = dto.SoundFile
	queue.Remark = dto.Remark

	return s.queueRepo.Update(ctx, queue)
}

// Delete deletes a queue.
func (s *QueueService) Delete(ctx context.Context, tenantID, id uint64) error {
	return s.queueRepo.Delete(ctx, tenantID, id)
}

// UpdateStatus updates a queue's status.
func (s *QueueService) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return s.queueRepo.UpdateStatus(ctx, tenantID, id, status)
}

// ListRealtime returns the real-time queue list.
func (s *QueueService) ListRealtime(ctx context.Context, tenantID uint64) ([]*model.Queue, error) {
	return s.queueRepo.ListRealtime(ctx, tenantID)
}
