package ports

import (
	"context"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
)

type ContractRepository interface {
	Create(ctx context.Context, c domain.Contract) (domain.Contract, error)
	GetByID(ctx context.Context, id int64) (domain.Contract, error)
	GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error)
	GetByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error)
	GetArchived(ctx context.Context, branchID int, page, pageSize int) ([]domain.Contract, int, error)
	GetArchivedByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error)
	RestoreContract(ctx context.Context, id int64) error
	SearchDashboard(ctx context.Context, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error)
	CheckCountry(ctx context.Context, name string) (bool, error)
	CheckCurrency(ctx context.Context, code string) (bool, error)
	Update(ctx context.Context, id int64, c domain.Contract) (domain.Contract, error)
	SoftDelete(ctx context.Context, id int64) error
	ResetApprovalStatus(ctx context.Context, id int64) error
}

type CounterpartyRepository interface {
	Create(ctx context.Context, c domain.Counterparty) (domain.Counterparty, error)
	CheckExistsInBranch(ctx context.Context, branchID int, llc string) (bool, error)
	CheckExistsByINN(ctx context.Context, inn string) (bool, error)
	GetByID(ctx context.Context, id int64) (domain.Counterparty, error)
	Update(ctx context.Context, id int64, c domain.Counterparty) (domain.Counterparty, error)
	SoftDelete(ctx context.Context, id int64) error
}

type BranchRepository interface {
	Create(ctx context.Context, b domain.Branch) (domain.Branch, error)
	GetByID(ctx context.Context, id int) (*domain.Branch, error)
	GetAll(ctx context.Context) ([]domain.Branch, error)
	Update(ctx context.Context, id int, name string) (*domain.Branch, error)
	SoftDelete(ctx context.Context, id int) error
	CheckExists(ctx context.Context, id int) (bool, error)
	CheckHasRelations(ctx context.Context, id int) (bool, error)
}

type AuthRepository interface {
	GetUserByLogin(ctx context.Context, login string) (*domain.User, error)
	GetUserByID(ctx context.Context, id int64) (*domain.User, error)
	GetAllUsers(ctx context.Context) ([]domain.User, error)
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	UpdateUser(ctx context.Context, id int64, role string, branchID int64) (domain.User, error)
	UpdateUserInfo(ctx context.Context, login, lastName, firstName, email string) error
	DeleteUser(ctx context.Context, id int64) error
	CreateAccessRequest(ctx context.Context, login, lastName, firstName, email string, branchID int64, role string) (domain.AccessRequest, error)
	GetRequestByID(ctx context.Context, requestID int64) (*domain.AccessRequest, error)
	GetPendingRequests(ctx context.Context) ([]domain.AccessRequest, error)
	GetAccessRequestsHistory(ctx context.Context) ([]domain.AccessRequest, error)
	ApproveRequest(ctx context.Context, requestID int64, reviewer string) (domain.User, error)
	RejectRequest(ctx context.Context, requestID int64, reviewer string) error
	SaveSession(ctx context.Context, token, login, lastName, firstName, email, role string, branchID int64, expiresAt time.Time) (domain.Session, error)
	GetSessionByToken(ctx context.Context, token string) (*domain.Session, error)
	SetAccessRequestSessionToken(ctx context.Context, requestID int64, token string) error
	DeleteSession(ctx context.Context, token string) error
	DeleteSessionsByLogin(ctx context.Context, login string) error
}

type InvoiceRepository interface {
	GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error)
	GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.InvoiceWithDetails, error)
	Create(ctx context.Context, inv domain.Invoice, contractCurrency string) (domain.Invoice, error)
	GetByID(ctx context.Context, id int64) (domain.Invoice, error)
	Update(ctx context.Context, id int64, inv domain.Invoice) (domain.Invoice, error)
	SoftDelete(ctx context.Context, id int64) error
	ResetApprovalStatus(ctx context.Context, id int64) error
}

type GTDRepository interface {
	Create(ctx context.Context, g domain.GTD) (domain.GTD, error)
	GetByID(ctx context.Context, id int64) (*domain.GTD, error)
	GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error)
	GetListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.GTD, error)
	GetByContractID(ctx context.Context, contractID int64) ([]domain.GTD, error)
	GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.GTD, error)
	Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error)
	SoftDelete(ctx context.Context, id int64) error
	ResetApprovalStatus(ctx context.Context, id int64) error
}

type PaymentOrderRepository interface {
	Create(ctx context.Context, po domain.PaymentOrder) (domain.PaymentOrder, error)
	GetByID(ctx context.Context, id int64) (*domain.PaymentOrder, error)
	GetByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.PaymentOrder, error)
	GetByContractID(ctx context.Context, contractID int64) ([]domain.PaymentOrder, error)
	Update(ctx context.Context, id int64, po domain.PaymentOrder) (*domain.PaymentOrder, error)
	SoftDelete(ctx context.Context, id int64) error
}

type GTDExtensionRepository interface {
	CreateRequest(ctx context.Context, req domain.GTDExtensionRequest) (domain.GTDExtensionRequest, error)
	GetByID(ctx context.Context, id int64) (*domain.GTDExtensionRequest, error)
	GetByGTDID(ctx context.Context, gtdID int64) ([]domain.GTDExtensionRequest, error)
	GetPendingRequests(ctx context.Context, branchID *int, page, pageSize int) ([]domain.GTDExtensionRequest, int, error)
	ReviewRequest(ctx context.Context, id int64, decision string, approvedDeadline *time.Time, comment, reviewer string) (*domain.GTDExtensionRequest, error)
}

type AdditionalAgreementRepository interface {
	Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error)
	GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error)
	Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	SoftDelete(ctx context.Context, id int64) error
	RestoreAdditionalAgreement(ctx context.Context, id int64) error
	ResetApprovalStatus(ctx context.Context, id int64) error
}

type ReportFilter struct {
	BranchID *int
	FromDate *time.Time
	ToDate   *time.Time
	Currency string
}

type ReportRepository interface {
	GetContractsReport(ctx context.Context, filter ReportFilter) (*dto.ContractsReportResponse, error)
	GetContractsExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.ContractExcelRow, string, error)
	GetInvoicesExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.InvoiceExcelRow, string, error)
	GetGTDExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.GTDExcelRow, string, error)
	GetAAExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.AAExcelRow, string, error)
	GetClientsExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.ClientExcelRow, error)
	GetPaymentOrdersExcelData(ctx context.Context, filter dto.ExcelReportFilter) ([]dto.PaymentOrderExcelRow, string, error)
	GetClientCurrencies(ctx context.Context, clientID int64) ([]string, error)
	GetClientConsolidatedReportData(ctx context.Context, clientID int64, filter dto.ExcelReportFilter) (*dto.ClientConsolidatedReportData, error)
	GetClientIDByINN(ctx context.Context, inn string) (int64, error)
}

type AuditLogFilter struct {
	UserLogin string
	Action    string
	Entity    string
	BranchID  *int64
	FromDate  *time.Time
	ToDate    *time.Time
	Limit     int
	Offset    int
}

type AuditLogRepository interface {
	Create(ctx context.Context, log domain.AuditLog) error
	List(ctx context.Context, filter AuditLogFilter) ([]domain.AuditLog, int64, error)
}

type TrashRepository interface {
	GetTrashItems(ctx context.Context, filter dto.TrashFilter) ([]dto.TrashItem, int, error)
	GetTrashItemByID(ctx context.Context, entityType string, id int64) (*dto.TrashItem, error)
	RestoreItem(ctx context.Context, entityType string, id int64) error
}

type PermissionRepository interface {
	GetCurrencyControlPermissions(ctx context.Context) ([]domain.CurrencyControlPermission, error)
	CheckCurrencyControlPermission(ctx context.Context, login string, action string) (bool, error)
	GrantCurrencyControlPermission(ctx context.Context, perm domain.CurrencyControlPermission) error
	RevokeCurrencyControlPermission(ctx context.Context, login string) error
}

type ApprovalRepository interface {
	SetCurrencyControlDecision(ctx context.Context, entityType string, id int64, decision, comment, reviewer string) (*dto.ApprovalItemResponse, error)
	SetComplianceDecision(ctx context.Context, entityType string, id int64, decision, comment, reviewer string) (*dto.ApprovalItemResponse, error)
	GetPendingApprovals(ctx context.Context, filter dto.PendingApprovalsFilter) ([]dto.ApprovalItemResponse, int, error)
	GetApprovalDetail(ctx context.Context, entityType string, id int64) (*dto.ApprovalItemResponse, error)
	ResetToPendingCurrencyControl(ctx context.Context, entityType string, id int64) error
}

type DocumentRepository interface {
	GetDocumentFileInfo(ctx context.Context, entityType string, id int64) (*domain.DocumentFileInfo, error)
}

type MyDocumentsRepository interface {
	GetMyDocuments(ctx context.Context, login string, filter dto.MyDocumentsFilter) ([]dto.MyDocumentItem, int, error)
}

