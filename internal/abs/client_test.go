package abs

import (
	"testing"

	"CurrencyControl/internal/domain"
)

// реальный фрагмент reportData из CBS (из примера пользователя)
const sampleReportData = `<MT94x><TITLE><S_CLI_PH1_NUM>992714475050</S_CLI_PH1_NUM><S_CLI_TYPE_NAME>Физические лица</S_CLI_TYPE_NAME><S_CLI_DEP_CODE>5000</S_CLI_DEP_CODE><S_CLI_CODE>5000.140641</S_CLI_CODE><S_CLI_SURNAME>Рачабов</S_CLI_SURNAME><S_CLI_NAME>Некрузчон</S_CLI_NAME><S_CLI_PATRONYMIC>Саймухторович</S_CLI_PATRONYMIC><S_CLI_LTN_SURNAME>Rajabov</S_CLI_LTN_SURNAME><S_CLI_LTN_NAME>Nekruzcon</S_CLI_LTN_NAME><S_CLI_LTN_PATRONYMIC>Saimuhtorovic</S_CLI_LTN_PATRONYMIC><S_CLI_TAX_CODE>987654321</S_CLI_TAX_CODE><S_CLI_IDENTDOC_NAME>Паспорт Республики Таджикистан</S_CLI_IDENTDOC_NAME><S_CLI_IDENTDOC_SERIES>a</S_CLI_IDENTDOC_SERIES><S_CLI_IDENTDOC_NUM>0708092</S_CLI_IDENTDOC_NUM><S_CLI_ADDRESS_NAME>ТОҶИКИСТОН, ДУШАНБЕ</S_CLI_ADDRESS_NAME></TITLE></MT94x>`

func TestParseReportData_Individual(t *testing.T) {
	res, err := parseReportData(sampleReportData, "987654321")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.INN != "987654321" {
		t.Errorf("INN: got %q, want %q", res.INN, "987654321")
	}
	if res.FullName != "Рачабов Некрузчон Саймухторович" {
		t.Errorf("FullName: got %q, want %q", res.FullName, "Рачабов Некрузчон Саймухторович")
	}
	if res.ClientType != domain.ClientTypeIndividual {
		t.Errorf("ClientType: got %q, want %q", res.ClientType, domain.ClientTypeIndividual)
	}
	if len(res.Phones) == 0 || res.Phones[0] != "992714475050" {
		t.Errorf("Phones: got %v, want [992714475050]", res.Phones)
	}
}

func TestParseReportData_MissingName(t *testing.T) {
	data := `<MT94x><TITLE><S_CLI_TAX_CODE>123</S_CLI_TAX_CODE></TITLE></MT94x>`
	_, err := parseReportData(data, "123")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestFilterEmpty(t *testing.T) {
	got := filterEmpty("Иванов", "", "  ", "Иван")
	if len(got) != 2 || got[0] != "Иванов" || got[1] != "Иван" {
		t.Errorf("filterEmpty: got %v", got)
	}
}

func TestUnique(t *testing.T) {
	got := unique("992123", "992123", "992456", "")
	if len(got) != 2 {
		t.Errorf("unique: expected 2 elements, got %v", got)
	}
}
