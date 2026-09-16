package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExportExcelReportValidation(t *testing.T) {
	handler := &ReportHandler{}
	req := httptest.NewRequest(http.MethodGet, "/api/reports/export", nil)
	rr := httptest.NewRecorder()

	handler.ExportExcelReport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request when type is missing, got %d", rr.Code)
	}
}
