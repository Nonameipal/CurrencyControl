package dto

import "time"

type ContractReportItem struct {
	ID               int64      `json:"id"`
	BranchID         *int       `json:"branch_id"`
	BranchName       string     `json:"branch_name"`
	ContractNumber   string     `json:"contract_number"`
	ContractDate     time.Time  `json:"contract_date"`
	DeliveryDate     *time.Time `json:"delivery_date,omitempty"`
	ContractEndDate  *time.Time `json:"contract_end_date,omitempty"`
	TotalAmount      float64    `json:"total_amount"`
	RemainingAmount  float64    `json:"remaining_amount"`
	ContractCurrency string     `json:"contract_currency"`
	Subject          string     `json:"subject"`
	InvoicesCount    int        `json:"invoices_count"`
	InvoicesAmount   float64    `json:"invoices_amount"`
	IsOverdue        bool       `json:"is_overdue"`
	CreatedBy        string     `json:"created_by"`
}

type ContractsReportResponse struct {
	TotalContracts        int                  `json:"total_contracts"`
	TotalAmountByCurrency map[string]float64   `json:"total_amount_by_currency"`
	TotalInvoicesCount    int                  `json:"total_invoices_count"`
	OverdueCount          int                  `json:"overdue_count"`
	Contracts             []ContractReportItem `json:"contracts"`
}

const (
	ReportTypeContracts            = "contracts"
	ReportTypeInvoices             = "invoices"
	ReportTypeGTD                  = "gtd"
	ReportTypeAdditionalAgreements = "additional_agreements"
	ReportTypeClients              = "clients"
	ReportTypeClientConsolidated   = "client_consolidated"
	ReportTypePaymentOrders        = "payment_orders"
)

type ReportTypeInfo struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ExcelReportFilter struct {
	ClientID  *int64
	ClientINN string
	BranchID  *int
	FromDate  *time.Time
	ToDate    *time.Time
	Currency  string
}

type ClientConsolidatedTransfer struct {
	InvoiceDate          string
	InvoiceAmount        float64
	GTDAmount            float64
	GTDNumber            string
	DiffAmount           float64
	ContractDeliveryTerm int
	ActualDeliveryTerm   int
	DiffDays             string
	Note                 string
}

type ClientConsolidatedAA struct {
	Number               string
	Date                 string
	EndDate              string
	ForeignCompany       string
	Country              string
	TotalAmount          float64
	Currency             string
	ParentContractNumber string
	Transfers            []ClientConsolidatedTransfer
}

type ClientConsolidatedContract struct {
	Number               string
	Date                 string
	EndDate              string
	ForeignCompany       string
	Country              string
	TotalAmount          float64
	Currency             string
	Transfers            []ClientConsolidatedTransfer
	AdditionalAgreements []ClientConsolidatedAA
}

type ClientConsolidatedReportData struct {
	ClientName string
	Contracts  []ClientConsolidatedContract
}

type ContractExcelRow struct {
	Number          string
	Date            string
	Subject         string
	Amount          float64
	Currency        string
	ReturnDays      string
	DeliveryDate    string
	ContractEndDate string
	ReceiverName    string
	ReceiverAccount string
	ReceiverCountry string
}

type InvoiceExcelRow struct {
	Number         string
	Date           string
	Amount         float64
	Currency       string
	HSCode         string
	PaymentPurpose string
}

type GTDExcelRow struct {
	Number     string
	Date       string
	Amount     float64
	Currency   string
	HSCode     string
	SenderName string
	Country    string
}

type AAExcelRow struct {
	Number       string
	DocType      string
	Date         string
	ContractNum  string
	Amount       float64
	Currency     string
	DeliveryDate string
	ReturnDays   string
	ExtendDateTo string
	Subject      string
}

type ClientExcelRow struct {
	ID             int64
	Name           string
	INN            string
	BranchName     string
	ContractsCount int
	TotalAmount    float64
	CreatedAt      string
}

type PaymentOrderExcelRow struct {
	OperationDate      string
	PaymentOrderNumber string
	Amount             float64
	Currency           string
	Payer              string
	ReceiverName       string
	ReceiverBank       string
	PaymentPurpose     string
	ReceiverCountry    string
	ContractNumber     string
	InvoiceNumber      string
	ValueDate          string
}
