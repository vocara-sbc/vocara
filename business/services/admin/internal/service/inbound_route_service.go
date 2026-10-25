package service

import (
	"context"
	"regexp"
	"time"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
	"vocara/pkg/eventbus"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// InboundRouteService 呼入接待配置服务
type InboundRouteService struct {
	routeRepo *repository.InboundRouteRepository
	queueRepo *repository.QueueRepository
	// eventPub 事件发布者（可选；发布后 callctrl 即时刷新呼入路由缓存，DID 即建即用）
	eventPub eventbus.Publisher
}

// NewInboundRouteService 创建呼入接待服务
func NewInboundRouteService() *InboundRouteService {
	return &InboundRouteService{
		routeRepo: repository.NewInboundRouteRepository(),
		queueRepo: repository.NewQueueRepository(),
	}
}

// SetEventPublisher 注入事件发布者（nil 时跳过发布，仅依赖 callctrl 定时刷新兜底）
func (s *InboundRouteService) SetEventPublisher(pub eventbus.Publisher) {
	s.eventPub = pub
}

// publishRouteChange 发布呼入路由变更事件（异步，失败仅记录日志不影响主流程）
func (s *InboundRouteService) publishRouteChange(tenantID uint64, action, businessNumber string) {
	if s.eventPub == nil {
		return
	}
	evt := eventbus.NewEvent(eventbus.EventInboundRouteChange, tenantID, "admin", map[string]interface{}{
		"action":          action,
		"did": businessNumber,
	})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := s.eventPub.Publish(ctx, evt); err != nil {
			logger.Warn("publish inbound_route.change failed",
				zap.String("action", action),
				zap.String("businessNumber", businessNumber),
				zap.Error(err))
		}
	}()
}

// InboundRouteSaveDTO 呼入接待保存
type InboundRouteSaveDTO struct {
	RouteName     string `json:"route_name"`
	RouteCode     string `json:"route_code"`
	DID string `json:"did"`
	WelcomeSound   string `json:"welcome_sound"`
	WaitSound      string `json:"wait_sound"`
	QueueID        uint64 `json:"queue_id"`
	SkillGroupID   uint64 `json:"skill_group_id"`
	WorkTimeType   int    `json:"work_time_type"`
	WorkTimeConfig string `json:"work_time_config"`
	HolidayIDs     string `json:"holiday_ids"`
	OffWorkType    int    `json:"off_work_type"`
	OffWorkValue   string `json:"off_work_value"`
	OffWorkSound   string `json:"off_work_sound"`
	Sort           int    `json:"sort"`
	Status         int    `json:"status"`
	Remark         string `json:"remark"`
}

// InboundRouteUpdateDTO 呼入接待更新
type InboundRouteUpdateDTO struct {
	ID             uint64 `json:"id"`
	RouteName     string `json:"route_name"`
	DID string `json:"did"`
	WelcomeSound   string `json:"welcome_sound"`
	WaitSound      string `json:"wait_sound"`
	QueueID        uint64 `json:"queue_id"`
	SkillGroupID   uint64 `json:"skill_group_id"`
	WorkTimeType   int    `json:"work_time_type"`
	WorkTimeConfig string `json:"work_time_config"`
	HolidayIDs     string `json:"holiday_ids"`
	OffWorkType    int    `json:"off_work_type"`
	OffWorkValue   string `json:"off_work_value"`
	OffWorkSound   string `json:"off_work_sound"`
	Sort           int    `json:"sort"`
	Status         int    `json:"status"`
	Remark         string `json:"remark"`
}

// InboundRouteItem 供 callctrl 拉取的呼入路由项（internal API 数据结构）
type InboundRouteItem struct {
	DID          string `json:"did"`
	QueueID      int64  `json:"queue_id"`
	SkillGroupID int64  `json:"skill_group_id"`
	TenantID     uint64 `json:"tenant_id"`
	RouteID      uint64 `json:"route_id"`
}

// ListInboundRoutes 全租户启用呼入接待的 DID 路由列表（internal API 用）
func (s *InboundRouteService) ListInboundRoutes(ctx context.Context) ([]*InboundRouteItem, error) {
	admits, err := s.routeRepo.ListInboundEnabled(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*InboundRouteItem, 0, len(admits))
	for _, a := range admits {
		items = append(items, &InboundRouteItem{
			DID:          a.DID,
			QueueID:      int64(a.QueueID),
			SkillGroupID: int64(a.SkillGroupID),
			TenantID:     a.TenantID,
			RouteID:      a.ID,
		})
	}
	return items, nil
}

// HolidaySaveDTO 节假日保存
type HolidaySaveDTO struct {
	HolidayName string `json:"holiday_name"`
	HolidayType int    `json:"holiday_type"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	TimeConfig  string `json:"time_config"`
	Status      int    `json:"status"`
	Remark      string `json:"remark"`
}

// HolidayUpdateDTO 节假日更新
type HolidayUpdateDTO struct {
	ID          uint64 `json:"id"`
	HolidayName string `json:"holiday_name"`
	HolidayType int    `json:"holiday_type"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	TimeConfig  string `json:"time_config"`
	Status      int    `json:"status"`
	Remark      string `json:"remark"`
}

// Page 分页查询
func (s *InboundRouteService) Page(ctx context.Context, tenantID uint64, q *repository.InboundRoutePageQuery) ([]*model.InboundRoute, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.routeRepo.PageQuery(ctx, tenantID, q)
}

// List 查询列表
func (s *InboundRouteService) List(ctx context.Context, tenantID uint64) ([]*model.InboundRoute, error) {
	return s.routeRepo.ListAll(ctx, tenantID)
}

// GetByID 获取详情
func (s *InboundRouteService) GetByID(ctx context.Context, tenantID, id uint64) (*model.InboundRoute, error) {
	return s.routeRepo.GetByID(ctx, tenantID, id)
}

// Create 创建
func (s *InboundRouteService) Create(ctx context.Context, tenantID uint64, dto *InboundRouteSaveDTO) (*model.InboundRoute, error) {
	if dto.RouteName == "" || dto.RouteCode == "" {
		return nil, errors.ErrParamMissing
	}

	exists, err := s.routeRepo.ExistsByCode(ctx, tenantID, dto.RouteCode, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.ErrAlreadyExists.WithDetail("配置编码已存在")
	}

	if err := s.validateDID(ctx, tenantID, dto.DID, 0); err != nil {
		return nil, err
	}

	// 检查队列是否存在
	var queueName string
	if dto.QueueID > 0 {
		queue, err := s.queueRepo.GetByID(ctx, tenantID, dto.QueueID)
		if err != nil {
			return nil, err
		}
		queueName = queue.QueueName
	}

	config := &model.InboundRoute{
		BaseModel: model.BaseModel{
			TenantID: tenantID,
		},
		RouteName:     dto.RouteName,
		RouteCode:     dto.RouteCode,
		DID: dto.DID,
		WelcomeSound:   dto.WelcomeSound,
		WaitSound:      dto.WaitSound,
		QueueID:        dto.QueueID,
		QueueName:      queueName,
		SkillGroupID:   dto.SkillGroupID,
		WorkTimeType:   dto.WorkTimeType,
		WorkTimeConfig: dto.WorkTimeConfig,
		HolidayIDs:     dto.HolidayIDs,
		OffWorkType:    dto.OffWorkType,
		OffWorkValue:   dto.OffWorkValue,
		OffWorkSound:   dto.OffWorkSound,
		Sort:           dto.Sort,
		Status:         dto.Status,
		Remark:         dto.Remark,
	}
	if config.Status == 0 {
		config.Status = 1
	}

	err = s.routeRepo.Create(ctx, config)
	if err != nil {
		return nil, err
	}

	s.publishRouteChange(tenantID, "create", config.DID)
	return config, nil
}

// Update 更新
func (s *InboundRouteService) Update(ctx context.Context, tenantID uint64, dto *InboundRouteUpdateDTO) error {
	if dto.ID == 0 {
		return errors.ErrParamMissing
	}

	if err := s.validateDID(ctx, tenantID, dto.DID, dto.ID); err != nil {
		return err
	}

	config, err := s.routeRepo.GetByID(ctx, tenantID, dto.ID)
	if err != nil {
		return err
	}

	// 如果队列变了，更新队列名称
	if dto.QueueID != config.QueueID {
		if dto.QueueID > 0 {
			queue, err := s.queueRepo.GetByID(ctx, tenantID, dto.QueueID)
			if err != nil {
				return err
			}
			config.QueueName = queue.QueueName
		} else {
			config.QueueName = ""
		}
	}

	config.RouteName = dto.RouteName
	config.DID = dto.DID
	config.WelcomeSound = dto.WelcomeSound
	config.WaitSound = dto.WaitSound
	config.QueueID = dto.QueueID
	config.SkillGroupID = dto.SkillGroupID
	config.WorkTimeType = dto.WorkTimeType
	config.WorkTimeConfig = dto.WorkTimeConfig
	config.HolidayIDs = dto.HolidayIDs
	config.OffWorkType = dto.OffWorkType
	config.OffWorkValue = dto.OffWorkValue
	config.OffWorkSound = dto.OffWorkSound
	config.Sort = dto.Sort
	config.Status = dto.Status
	config.Remark = dto.Remark

	if err := s.routeRepo.Update(ctx, config); err != nil {
		return err
	}
	s.publishRouteChange(tenantID, "update", config.DID)
	return nil
}

// businessNumberPattern 呼入接入号（DID）格式：4~12 位纯数字。
// 内联测试接入号 4~6 位，运营商 DID 8~12 位；路由与号码位数无关，由本配置决定。
var businessNumberPattern = regexp.MustCompile(`^\d{4,12}$`)

// validateDID 校验业务号码（DID）格式与唯一性；为空则跳过（仅作非 DID 接待配置时允许）
func (s *InboundRouteService) validateDID(ctx context.Context, tenantID uint64, number string, excludeID uint64) error {
	if number == "" {
		return nil
	}
	if !businessNumberPattern.MatchString(number) {
		return errors.ErrParamInvalid.WithDetail("业务号码格式错误：须为 4~12 位纯数字")
	}
	exists, err := s.routeRepo.ExistsDID(ctx, tenantID, number, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return errors.ErrAlreadyExists.WithDetail("业务号码 " + number + " 已被其他呼入接待配置占用")
	}
	return nil
}

// Delete 删除
func (s *InboundRouteService) Delete(ctx context.Context, tenantID, id uint64) error {
	// 先取业务号码用于事件通知（删除后路由缓存需立即移除该 DID）
	var businessNumber string
	if config, err := s.routeRepo.GetByID(ctx, tenantID, id); err == nil && config != nil {
		businessNumber = config.DID
	}
	if err := s.routeRepo.Delete(ctx, tenantID, id); err != nil {
		return err
	}
	s.publishRouteChange(tenantID, "delete", businessNumber)
	return nil
}

// UpdateStatus 更新状态
func (s *InboundRouteService) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	var businessNumber string
	if config, err := s.routeRepo.GetByID(ctx, tenantID, id); err == nil && config != nil {
		businessNumber = config.DID
	}
	if err := s.routeRepo.UpdateStatus(ctx, tenantID, id, status); err != nil {
		return err
	}
	s.publishRouteChange(tenantID, "status", businessNumber)
	return nil
}

// ===== 节假日 =====

// HolidayPage 分页查询节假日
func (s *InboundRouteService) HolidayPage(ctx context.Context, tenantID uint64, q *repository.HolidayPageQuery) ([]*model.HolidaySchedule, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.routeRepo.PageQueryHoliday(ctx, tenantID, q)
}

// HolidayList 查询所有节假日
func (s *InboundRouteService) HolidayList(ctx context.Context, tenantID uint64) ([]*model.HolidaySchedule, error) {
	return s.routeRepo.ListAllHolidays(ctx, tenantID)
}

// HolidayGet 获取节假日详情
func (s *InboundRouteService) HolidayGet(ctx context.Context, tenantID, id uint64) (*model.HolidaySchedule, error) {
	return s.routeRepo.GetHolidayByID(ctx, tenantID, id)
}

// HolidayCreate 创建节假日
func (s *InboundRouteService) HolidayCreate(ctx context.Context, tenantID uint64, dto *HolidaySaveDTO) (*model.HolidaySchedule, error) {
	if dto.HolidayName == "" {
		return nil, errors.ErrParamMissing.WithDetail("节假日名称不能为空")
	}

	startDate, err := parseDate(dto.StartDate)
	if err != nil {
		return nil, errors.ErrParamInvalid.WithDetail("开始日期格式错误")
	}
	endDate, err := parseDate(dto.EndDate)
	if err != nil {
		return nil, errors.ErrParamInvalid.WithDetail("结束日期格式错误")
	}

	holiday := &model.HolidaySchedule{
		BaseModel: model.BaseModel{
			TenantID: tenantID,
		},
		HolidayName: dto.HolidayName,
		HolidayType: dto.HolidayType,
		StartDate:   startDate,
		EndDate:     endDate,
		TimeConfig:  dto.TimeConfig,
		Status:      dto.Status,
		Remark:      dto.Remark,
	}
	if holiday.HolidayType == 0 {
		holiday.HolidayType = 1
	}
	if holiday.Status == 0 {
		holiday.Status = 1
	}

	err = s.routeRepo.CreateHoliday(ctx, holiday)
	if err != nil {
		return nil, err
	}
	return holiday, nil
}

// HolidayUpdate 更新节假日
func (s *InboundRouteService) HolidayUpdate(ctx context.Context, tenantID uint64, dto *HolidayUpdateDTO) error {
	if dto.ID == 0 {
		return errors.ErrParamMissing
	}

	holiday, err := s.routeRepo.GetHolidayByID(ctx, tenantID, dto.ID)
	if err != nil {
		return err
	}

	if dto.StartDate != "" {
		startDate, err := parseDate(dto.StartDate)
		if err != nil {
			return errors.ErrParamInvalid.WithDetail("开始日期格式错误")
		}
		holiday.StartDate = startDate
	}
	if dto.EndDate != "" {
		endDate, err := parseDate(dto.EndDate)
		if err != nil {
			return errors.ErrParamInvalid.WithDetail("结束日期格式错误")
		}
		holiday.EndDate = endDate
	}

	holiday.HolidayName = dto.HolidayName
	holiday.HolidayType = dto.HolidayType
	holiday.TimeConfig = dto.TimeConfig
	holiday.Status = dto.Status
	holiday.Remark = dto.Remark

	return s.routeRepo.UpdateHoliday(ctx, holiday)
}

// HolidayDelete 删除节假日
func (s *InboundRouteService) HolidayDelete(ctx context.Context, tenantID, id uint64) error {
	return s.routeRepo.DeleteHoliday(ctx, tenantID, id)
}

// parseDate 解析日期
func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006/01/02",
	}
	for _, format := range formats {
		if t, err := time.ParseInLocation(format, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.ErrParamInvalid
}
