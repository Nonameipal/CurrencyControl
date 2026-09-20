package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type reportService struct {
	repo ports.ReportRepository
}

func NewReportService(repo ports.ReportRepository) ports.ReportService {
	return &reportService{repo: repo}
}

func (s *reportService) GetContractsReport(ctx context.Context, userRole string, userBranchID int64, filter ports.ReportFilter) (*dto.ContractsReportResponse, error) {
	switch userRole {
	case domain.RoleBranchHead:
		if filter.BranchID != nil && int64(*filter.BranchID) != userBranchID {
			return nil, errors.New("начальник подразделения имеет доступ к формированию отчетности только своего подразделения")
		}
		branchInt := int(userBranchID)
		filter.BranchID = &branchInt

	case domain.RoleCurrencyControl, domain.RoleCurrencyController, domain.RoleCompliance, domain.RoleInternalAudit, domain.RoleAdmin:

	default:
		return nil, errors.New("недостаточно прав для формирования отчетности")
	}

	return s.repo.GetContractsReport(ctx, filter)
}

func (s *reportService) GetReportTypes() []dto.ReportTypeInfo {
	return []dto.ReportTypeInfo{
		{
			Type:        dto.ReportTypeContracts,
			Name:        "Отчет по контрактам клиента",
		},
		{
			Type:        dto.ReportTypeInvoices,
			Name:        "Отчет по инвойсам клиента",
		},
		{
			Type:        dto.ReportTypeGTD,
			Name:        "Отчет по ГТД клиента",
		},
		{
			Type:        dto.ReportTypeAdditionalAgreements,
			Name:        "Отчет по дополнительным соглашениям клиента",
		},
		{
			Type:        dto.ReportTypeClients,
			Name:        "Отчет по клиентам банка",
		},
		{
			Type: dto.ReportTypeClientConsolidated,
			Name: "Общий отчет по клиенту",
		},
		{
			Type: dto.ReportTypePaymentOrders,
			Name: "Отчет по платежным поручениям",
		},
	}
}

func (s *reportService) GetClientCurrencies(ctx context.Context, clientID int64) ([]string, error) {
	return s.repo.GetClientCurrencies(ctx, clientID)
}

func (s *reportService) ExportExcelReport(ctx context.Context, userRole string, userBranchID int64, reportType string, filter dto.ExcelReportFilter) ([]byte, string, error) {
	switch userRole {
	case domain.RoleBranchHead:
		if filter.BranchID != nil && int64(*filter.BranchID) != userBranchID {
			return nil, "", errors.New("начальник подразделения имеет доступ к формированию отчетности только своего подразделения")
		}
		branchInt := int(userBranchID)
		filter.BranchID = &branchInt

	case domain.RoleOperator, domain.RoleCurrencyControl, domain.RoleCurrencyController, domain.RoleCompliance, domain.RoleInternalAudit, domain.RoleAdmin:

	default:
		return nil, "", errors.New("недостаточно прав для формирования отчетности")
	}

	dateStr := time.Now().Format("20060102_1504")

	if strings.TrimSpace(filter.ClientINN) != "" && (filter.ClientID == nil || *filter.ClientID <= 0) {
		id, err := s.repo.GetClientIDByINN(ctx, filter.ClientINN)
		if err != nil || id <= 0 {
			return nil, "", fmt.Errorf("клиент с ИНН '%s' не найден", filter.ClientINN)
		}
		filter.ClientID = &id
	}

	switch reportType {
	case dto.ReportTypeContracts:
		rows, clientName, err := s.repo.GetContractsExcelData(ctx, filter)
		if err != nil {
			return nil, "", err
		}
		fileBytes, err := GenerateContractsExcel(rows, clientName)
		if err != nil {
			return nil, "", err
		}
		return fileBytes, fmt.Sprintf("report_contracts_%s.xlsx", dateStr), nil

	case dto.ReportTypeInvoices:
		rows, clientName, err := s.repo.GetInvoicesExcelData(ctx, filter)
		if err != nil {
			return nil, "", err
		}
		fileBytes, err := GenerateInvoicesExcel(rows, clientName)
		if err != nil {
			return nil, "", err
		}
		return fileBytes, fmt.Sprintf("report_invoices_%s.xlsx", dateStr), nil

	case dto.ReportTypeGTD:
		rows, clientName, err := s.repo.GetGTDExcelData(ctx, filter)
		if err != nil {
			return nil, "", err
		}
		fileBytes, err := GenerateGTDExcel(rows, clientName)
		if err != nil {
			return nil, "", err
		}
		return fileBytes, fmt.Sprintf("report_gtd_%s.xlsx", dateStr), nil

	case dto.ReportTypeAdditionalAgreements:
		rows, clientName, err := s.repo.GetAAExcelData(ctx, filter)
		if err != nil {
			return nil, "", err
		}
		fileBytes, err := GenerateAAExcel(rows, clientName)
		if err != nil {
			return nil, "", err
		}
		return fileBytes, fmt.Sprintf("report_additional_agreements_%s.xlsx", dateStr), nil

	case dto.ReportTypeClients:
		rows, err := s.repo.GetClientsExcelData(ctx, filter)
		if err != nil {
			return nil, "", err
		}
		fileBytes, err := GenerateClientsExcel(rows)
		if err != nil {
			return nil, "", err
		}
		return fileBytes, fmt.Sprintf("report_clients_%s.xlsx", dateStr), nil

	case dto.ReportTypeClientConsolidated:
		if filter.ClientID == nil || *filter.ClientID <= 0 {
			return nil, "", errors.New("ИНН или ID клиента обязателен для общего отчета по клиенту")
		}
		data, err := s.repo.GetClientConsolidatedReportData(ctx, *filter.ClientID, filter)
		if err != nil {
			return nil, "", err
		}
		fileBytes, err := GenerateClientConsolidatedExcel(data)
		if err != nil {
			return nil, "", err
		}
		identifier := fmt.Sprintf("%d", *filter.ClientID)
		if strings.TrimSpace(filter.ClientINN) != "" {
			identifier = strings.TrimSpace(filter.ClientINN)
		}
		return fileBytes, fmt.Sprintf("report_client_consolidated_%s_%s.xlsx", identifier, dateStr), nil

	case dto.ReportTypePaymentOrders:
		rows, clientName, err := s.repo.GetPaymentOrdersExcelData(ctx, filter)
		if err != nil {
			return nil, "", err
		}
		fileBytes, err := GeneratePaymentOrdersExcel(rows, clientName)
		if err != nil {
			return nil, "", err
		}
		return fileBytes, fmt.Sprintf("report_payment_orders_%s.xlsx", dateStr), nil

	default:
		return nil, "", fmt.Errorf("неизвестный тип отчета: %s", reportType)
	}
}

func (s *reportService) SaveTemplate(reportType string, data []byte) error {
	return SaveReportTemplate(reportType, data)
}

func (s *reportService) GetTemplate(reportType string) ([]byte, error) {
	return GetReportTemplate(reportType)
}
