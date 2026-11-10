package repository

import (
	"fmt"
	"strings"
)

func BuildTrashSelect(alias, entityType, entityName, numberField, dateField, amountField, currencyField string) string {
	return fmt.Sprintf(`
		%s.id, '%s' AS entity_type, %s AS entity_name,
		COALESCE(%s, '') AS number, %s AS document_date,
		COALESCE(%s, 0) AS amount, COALESCE(%s, '') AS currency,
		COALESCE(%s.document_path, '') AS document_path,
		c.client_id, COALESCE(cp.llc, '') AS client_name,
		%s AS contract_id, COALESCE(c.contract_number, '') AS contract_number,
		COALESCE(%s.created_by, '') AS created_by, COALESCE(%s.deleted_by, '') AS deleted_by, %s.deleted_at,
		COALESCE(c.branch_id, cp.branch_id, 0) AS branch_id
	`, alias, entityType, entityName, numberField, dateField, amountField, currencyField, alias,
		func() string {
			if alias == "c" {
				return "c.id"
			}
			return alias + ".contract_id"
		}(),
		alias, alias, alias)
}

func BuildDocumentSelectBase(alias, entityType, numberField, dateField, subjectField, amountField, currencyField string, includeBranchID bool) string {
	branchIDPart := ""
	if includeBranchID {
		branchIDPart = "c.branch_id, "
	}

	entityTypeExpr := fmt.Sprintf("'%s'", entityType)
	if strings.Contains(entityType, ".") || strings.Contains(entityType, "(") {
		entityTypeExpr = entityType
	}
	
	if alias == "c" || alias == "i" || alias == "aa" || alias == "a" {
		return fmt.Sprintf(`
		%s AS entity_type, %s.id AS entity_id, %sCOALESCE(b.name, '') AS branch_name,
		%s AS document_number, %s.document_path, %s AS document_date, COALESCE(%s, '') AS subject,
		%s AS amount, %s AS currency, COALESCE(cp.llc, '') AS counterparty_name,
		'approved' AS approval_status,
		'' AS currency_control_decision,
		'' AS currency_control_comment,
		'' AS currency_control_reviewed_by,
		NULL::timestamptz AS currency_control_reviewed_at,
		'' AS compliance_decision,
		'' AS compliance_comment,
		'' AS compliance_reviewed_by,
		NULL::timestamptz AS compliance_reviewed_at,
		'' AS rejection_reason,
		COALESCE(%s.created_by, '') AS created_by, %s.created_at
	`,
			entityTypeExpr, alias, branchIDPart,
			numberField, alias, dateField, subjectField,
			amountField, currencyField,
			alias, alias)
	}

	return fmt.Sprintf(`
		%s AS entity_type, %s.id AS entity_id, %sCOALESCE(b.name, '') AS branch_name,
		%s AS document_number, %s.document_path, %s AS document_date, COALESCE(%s, '') AS subject,
		%s AS amount, %s AS currency, COALESCE(cp.llc, '') AS counterparty_name,
		COALESCE(%s.approval_status, 'pending_currency_control') AS approval_status,
		COALESCE(%s.currency_control_decision, '') AS currency_control_decision,
		COALESCE(%s.currency_control_comment, '') AS currency_control_comment,
		COALESCE(%s.currency_control_reviewed_by, '') AS currency_control_reviewed_by,
		%s.currency_control_reviewed_at,
		COALESCE(%s.compliance_decision, '') AS compliance_decision,
		COALESCE(%s.compliance_comment, '') AS compliance_comment,
		COALESCE(%s.compliance_reviewed_by, '') AS compliance_reviewed_by,
		%s.compliance_reviewed_at,
		COALESCE(%s.rejection_reason, '') AS rejection_reason,
		COALESCE(%s.created_by, '') AS created_by, %s.created_at
	`,
		entityTypeExpr, alias, branchIDPart,
		numberField, alias, dateField, subjectField,
		amountField, currencyField,
		alias,
		alias,
		alias,
		alias,
		alias,
		alias,
		alias,
		alias,
		alias,
		alias,
		alias, alias)
}

func BuildMyDocumentsSelect(alias, entityType, numberField, dateField, subjectField, amountField, currencyField string) string {
	return BuildDocumentSelectBase(alias, entityType, numberField, dateField, subjectField, amountField, currencyField, false)
}

func BuildApprovalSelect(alias, entityType, numberField, dateField, subjectField, amountField, currencyField string) string {
	return BuildDocumentSelectBase(alias, entityType, numberField, dateField, subjectField, amountField, currencyField, true)
}
