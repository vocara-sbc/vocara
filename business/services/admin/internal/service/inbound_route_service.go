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

// InboundRouteService provides inbound reception configuration operations.
type InboundRouteService struct {
	routeRepo *repository.InboundRouteRepository
	queueRepo *repository.QueueRepository
	// eventPub optionally publishes events so callctrl refreshes its inbound route cache immediately.
	eventPub eventbus.Publisher
}

// NewInboundRouteService creates an inbound reception service.
func NewInboundRouteService() *InboundRouteService {
	return &InboundRouteService{
		routeRepo: repository.NewInboundRouteRepository(),
		queueRepo: repository.NewQueueRepository(),
	}
}

// SetEventPublisher sets the event publisher; nil disables publishing and relies on callctrl's periodic refresh.
func (s *InboundRouteService) SetEventPublisher(pub eventbus.Publisher) {
	s.eventPub = pub
}

// publishRouteChange asynchronously publishes an inbound route change and logs failures without interrupting the main flow.
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

// InboundRouteSaveDTO contains an inbound reception creation request.
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

// InboundRouteUpdateDTO contains an inbound reception update request.
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

// InboundRouteItem is the internal API inbound route structure returned to callctrl.
type InboundRouteItem struct {
	DID          string `json:"did"`
	QueueID      int64  `json:"queue_id"`
	SkillGroupID int64  `json:"skill_group_id"`
	TenantID     uint64 `json:"tenant_id"`
	RouteID      uint64 `json:"route_id"`
}

// ListInboundRoutes returns enabled inbound reception DID routes across all tenants for the internal API.
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

// HolidaySaveDTO contains a holiday creation request.
type HolidaySaveDTO struct {
	HolidayName string `json:"holiday_name"`
	HolidayType int    `json:"holiday_type"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	TimeConfig  string `json:"time_config"`
	Status      int    `json:"status"`
	Remark      string `json:"remark"`
}

// HolidayUpdateDTO contains a holiday update request.
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

// Page executes a paginated inbound reception query.
func (s *InboundRouteService) Page(ctx context.Context, tenantID uint64, q *repository.InboundRoutePageQuery) ([]*model.InboundRoute, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.routeRepo.PageQuery(ctx, tenantID, q)
}

// List returns all inbound reception configurations.
func (s *InboundRouteService) List(ctx context.Context, tenantID uint64) ([]*model.InboundRoute, error) {
	return s.routeRepo.ListAll(ctx, tenantID)
}

// GetByID returns inbound reception details by ID.
func (s *InboundRouteService) GetByID(ctx context.Context, tenantID, id uint64) (*model.InboundRoute, error) {
	return s.routeRepo.GetByID(ctx, tenantID, id)
}

// Create creates an inbound reception configuration.
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

	// Check whether the queue exists.
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

// Update updates an inbound reception configuration.
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

	// Update the queue name when the queue changes.
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

// businessNumberPattern matches a DID containing 4 to 12 digits.
// Test access numbers contain 4 to 6 digits and carrier DIDs contain 8 to 12; this configuration determines routing independently of number length.
var businessNumberPattern = regexp.MustCompile(`^\d{4,12}$`)

// validateDID validates DID format and uniqueness; an empty value is allowed for non-DID reception configurations.
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

// Delete deletes an inbound reception configuration.
func (s *InboundRouteService) Delete(ctx context.Context, tenantID, id uint64) error {
	// Read the DID for the event so the route cache can remove it after deletion.
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

// UpdateStatus updates an inbound reception configuration's status.
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

// ===== Holidays =====

// HolidayPage executes a paginated holiday query.
func (s *InboundRouteService) HolidayPage(ctx context.Context, tenantID uint64, q *repository.HolidayPageQuery) ([]*model.HolidaySchedule, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.routeRepo.PageQueryHoliday(ctx, tenantID, q)
}

// HolidayList returns all holidays.
func (s *InboundRouteService) HolidayList(ctx context.Context, tenantID uint64) ([]*model.HolidaySchedule, error) {
	return s.routeRepo.ListAllHolidays(ctx, tenantID)
}

// HolidayGet returns holiday details by ID.
func (s *InboundRouteService) HolidayGet(ctx context.Context, tenantID, id uint64) (*model.HolidaySchedule, error) {
	return s.routeRepo.GetHolidayByID(ctx, tenantID, id)
}

// HolidayCreate creates a holiday.
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

// HolidayUpdate updates a holiday.
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

// HolidayDelete deletes a holiday.
func (s *InboundRouteService) HolidayDelete(ctx context.Context, tenantID, id uint64) error {
	return s.routeRepo.DeleteHoliday(ctx, tenantID, id)
}

// parseDate parses a date using a supported format.
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
