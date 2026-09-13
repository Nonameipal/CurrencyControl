package excel_test

import (
	"bytes"
	"testing"

	"CurrencyControl/pkg/excel"

	"github.com/xuri/excelize/v2"
)

type SampleRow struct {
	ID     int
	Name   string
	Amount float64
	Date   string
}

func TestGenerateTable(t *testing.T) {
	cols := []excel.ColumnDef[SampleRow]{
		{Header: "ID", Width: 10, Type: excel.CellInt, GetValue: func(r SampleRow) any { return r.ID }},
		{Header: "Имя", Width: 20, Type: excel.CellText, GetValue: func(r SampleRow) any { return r.Name }},
		{Header: "Сумма", Width: 15, Type: excel.CellAmount, GetValue: func(r SampleRow) any { return r.Amount }},
		{Header: "Дата", Width: 12, Type: excel.CellDate, GetValue: func(r SampleRow) any { return r.Date }},
	}

	rows := []SampleRow{
		{ID: 1, Name: "Тест 1", Amount: 1234.56, Date: "01.01.2026"},
		{ID: 2, Name: "Тест 2", Amount: 7890.12, Date: "02.01.2026"},
	}

	data, err := excel.GenerateTable(excel.TableConfig[SampleRow]{
		SheetName: "Тест",
		Title:     "Тестовая Таблица",
		Columns:   cols,
		Rows:      rows,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty excel bytes")
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("cannot open generated excel: %v", err)
	}
	defer f.Close()

	sheet := f.GetSheetList()[0]
	title, _ := f.GetCellValue(sheet, "A1")
	if title != "Тестовая Таблица" {
		t.Errorf("expected title 'Тестовая Таблица', got %q", title)
	}

	h1, _ := f.GetCellValue(sheet, "A2")
	if h1 != "ID" {
		t.Errorf("expected header 'ID', got %q", h1)
	}

	h2, _ := f.GetCellValue(sheet, "B2")
	if h2 != "Имя" {
		t.Errorf("expected header 'Имя', got %q", h2)
	}

	v1, _ := f.GetCellValue(sheet, "A3")
	if v1 != "1" {
		t.Errorf("expected row 1 ID '1', got %q", v1)
	}

	v2, _ := f.GetCellValue(sheet, "B3")
	if v2 != "Тест 1" {
		t.Errorf("expected row 1 Name 'Тест 1', got %q", v2)
	}
}

func TestGenerateTable_EmptyRows(t *testing.T) {
	cols := []excel.ColumnDef[SampleRow]{
		{Header: "ID", Width: 10, Type: excel.CellInt, GetValue: func(r SampleRow) any { return r.ID }},
	}

	data, err := excel.GenerateTable(excel.TableConfig[SampleRow]{
		SheetName: "Пусто",
		Title:     "Заголовок",
		Columns:   cols,
		Rows:      nil,
	})
	if err != nil {
		t.Fatalf("unexpected error on nil rows: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty bytes")
	}
}
