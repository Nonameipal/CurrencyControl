package ports

import (
	"context"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
)

type LoginStatus string

const (
	StatusActive LoginStatus = "active"
	StatusNoRole LoginStatus = "no_role"
)

type LoginResult struct {
	Status              LoginStatus     `json:"status"`
	AccessToken         string          `json:"access_token,omitempty"`
	RefreshToken        string          `json:"refresh_token,omitempty"`
	AccessTokenExpires  time.Time       `json:"access_token_expires_at,omitempty"`
	RefreshTokenExpires time.Time       `json:"refresh_token_expires_at,omitempty"`
	Login               string          `json:"login,omitempty"`
	Message             string          `json:"message"`
	Session             *domain.Session `json:"session,omitempty"`
}

type ContractService interface {
	Create(ctx context.Context, login string, input domain.Contract) (domain.Contract, error)
	GetByID(ctx context.Context, login string, id int64) (domain.Contract, error)
	GetByClientID(ctx context.Context, login string, clientID int64) ([]domain.Contract, error)
	GetArchived(ctx context.Context, branchID int, page, pageSize int) ([]domain.Contract, int, error)
	GetArchivedByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error)
	RestoreContract(ctx context.Context, id int64) error
	SearchDashboard(ctx context.Context, login string, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error)
	CheckCountry(ctx context.Context, name string) (bool, error)
	CheckCurrency(ctx context.Context, code string) (bool, error)
	GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error)
	Update(ctx context.Context, id int64, c domain.Contract) (domain.Contract, error)
	SoftDelete(ctx context.Context, id int64) error
}

type CounterpartyService interface {
	// CreateFromABS — единый маршрут создания ЧДММ: сначала обязательный запрос в АБС по ИНН,
	// при успехе — автосоздание карточки контрагента с данными из АБС + название из запроса.
	CreateFromABS(ctx context.Context, login, llc, inn string, branchID int) (domain.Counterparty, error)
	CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error)
	CheckExistsByINN(ctx context.Context, inn string) (bool, error)
	GetByID(ctx context.Context, id int64) (domain.Counterparty, error)
	Update(ctx context.Context, id int64, input domain.Counterparty) (domain.Counterparty, error)
	SoftDelete(ctx context.Context, id int64) error
}

type BranchService interface {
	Create(ctx context.Context, login string, req dto.CreateBranchRequest) (domain.Branch, error)
	GetByID(ctx context.Context, id int) (*domain.Branch, error)
	GetAll(ctx context.Context) ([]domain.Branch, error)
	Update(ctx context.Context, id int, req dto.UpdateBranchRequest) (*domain.Branch, error)
	Delete(ctx context.Context, id int) error
}

type AuthService interface {
	Login(ctx context.Context, login, password string) (LoginResult, error)
	Refresh(ctx context.Context, refreshToken string) (LoginResult, error)
	RequestAccess(ctx context.Context, login string, branchID int64, role string) (domain.AccessRequest, error)
	GetRequestStatus(ctx context.Context, requestID int64) (*domain.AccessRequest, error)
	GetPendingRequests(ctx context.Context) ([]domain.AccessRequest, error)
	GetAccessRequestsHistory(ctx context.Context) ([]domain.AccessRequest, error)
	ApproveRequest(ctx context.Context, requestID int64, reviewer string) (LoginResult, error)
	RejectRequest(ctx context.Context, requestID int64, reviewer string) error
	GetAllUsers(ctx context.Context) ([]domain.User, error)
	CreateUser(ctx context.Context, req domain.CreateUserRequest) (domain.User, error)
	UpdateUser(ctx context.Context, id int64, req domain.UpdateUserRequest) (domain.User, error)
	DeleteUser(ctx context.Context, id int64) error
	ValidateSession(ctx context.Context, token string) (*domain.Session, error)
	Logout(ctx context.Context, token string) error
}

type InvoiceService interface {
	GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error)
	GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.InvoiceWithDetails, error)
	Create(ctx context.Context, inv domain.Invoice, contractCurrency string) (domain.Invoice, error)
	GetByID(ctx context.Context, id int64) (domain.Invoice, error)
	Update(ctx context.Context, id int64, inv domain.Invoice) (domain.Invoice, error)
	SoftDelete(ctx context.Context, id int64) error
}

type GTDService interface {
	Create(ctx context.Context, g domain.GTD) (domain.GTD, error)
	GetByID(ctx context.Context, id int64) (*domain.GTD, error)
	GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error)
	GetListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.GTD, error)
	GetByContractID(ctx context.Context, contractID int64) ([]domain.GTD, error)
	GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.GTD, error)
	Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error)
	SoftDelete(ctx context.Context, id int64) error
}

type PaymentOrderService interface {
	Create(ctx context.Context, po domain.PaymentOrder) (domain.PaymentOrder, error)
	GetByID(ctx context.Context, id int64) (*domain.PaymentOrder, error)
	GetByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.PaymentOrder, error)
	GetByContractID(ctx context.Context, contractID int64) ([]domain.PaymentOrder, error)
	Update(ctx context.Context, id int64, po domain.PaymentOrder) (*domain.PaymentOrder, error)
	SoftDelete(ctx context.Context, id int64) error
}

type GTDExtensionService interface {
	CreateRequest(ctx context.Context, login, role string, gtdID int64, requestedDeadline time.Time, documentPath string) (domain.GTDExtensionRequest, error)
	GetByID(ctx context.Context, id int64) (*domain.GTDExtensionRequest, error)
	GetByGTDID(ctx context.Context, gtdID int64) ([]domain.GTDExtensionRequest, error)
	GetPendingRequests(ctx context.Context, branchID *int, page, pageSize int) ([]domain.GTDExtensionRequest, int, error)
	ReviewRequest(ctx context.Context, role, login string, id int64, decision string, approvedDeadline *time.Time, comment string) (*domain.GTDExtensionRequest, error)
}

type AdditionalAgreementService interface {
	Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error)
	GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error)
	Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	SoftDelete(ctx context.Context, id int64) error
	RestoreAdditionalAgreement(ctx context.Context, id int64) error
}

type ReportService interface {
	GetContractsReport(ctx context.Context, userRole string, userBranchID int64, filter ReportFilter) (*dto.ContractsReportResponse, error)
	GetReportTypes() []dto.ReportTypeInfo
	GetClientCurrencies(ctx context.Context, clientID int64) ([]string, error)
	ExportExcelReport(ctx context.Context, userRole string, userBranchID int64, reportType string, filter dto.ExcelReportFilter) ([]byte, string, error)
	SaveTemplate(reportType string, data []byte) error
	GetTemplate(reportType string) ([]byte, error)
}

type AuditLogService interface {
	Log(ctx context.Context, login, role string, branchID *int64, action, entity string, entityID *int64, details, ip string)
	List(ctx context.Context, filter AuditLogFilter) ([]domain.AuditLog, int64, error)
}

type TrashService interface {
	GetTrashList(ctx context.Context, filter dto.TrashFilter) ([]dto.TrashItem, int, error)
	GetTrashItem(ctx context.Context, entityType string, id int64) (*dto.TrashItem, error)
	GetFilePath(ctx context.Context, entityType string, id int64) (string, error)
	RestoreItem(ctx context.Context, entityType string, id int64) error
}

type PermissionService interface {
	GetCurrencyControlPermissions(ctx context.Context) ([]domain.CurrencyControlPermission, error)
	CanEditFiles(ctx context.Context, role, login string) bool
	CanDeleteFiles(ctx context.Context, role, login string) bool
	GrantCurrencyControlPermission(ctx context.Context, perm domain.CurrencyControlPermission) error
	RevokeCurrencyControlPermission(ctx context.Context, login string) error
}

type ApprovalService interface {
	ReviewCurrencyControl(ctx context.Context, role, login string, entityType string, id int64, req dto.CurrencyControlDecisionRequest) (*dto.ApprovalItemResponse, error)
	ReviewCompliance(ctx context.Context, role, login string, entityType string, id int64, req dto.ComplianceDecisionRequest) (*dto.ApprovalItemResponse, error)
	GetPendingApprovals(ctx context.Context, role string, userBranchID int64, filter dto.PendingApprovalsFilter) (*dto.PendingApprovalsResponse, error)
	GetApprovalDetail(ctx context.Context, entityType string, id int64) (*dto.ApprovalItemResponse, error)
}
