package service

// import (
// 	"context"
// 	"testing"

// 	"CurrencyControl/internal/abs"
// 	"CurrencyControl/internal/domain"
// )

// type mockCounterpartyRepo struct {
// 	checkExistsResult bool
// 	created           domain.Counterparty
// }

// func (m *mockCounterpartyRepo) CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error) {
// 	return false, nil
// }
// func (m *mockCounterpartyRepo) CheckExistsByINN(ctx context.Context, inn string) (bool, error) {
// 	return m.checkExistsResult, nil
// }
// func (m *mockCounterpartyRepo) Create(ctx context.Context, c domain.Counterparty) (domain.Counterparty, error) {
// 	m.created = c
// 	c.ID = 123
// 	return c, nil
// }
// func (m *mockCounterpartyRepo) GetByID(ctx context.Context, id int64) (domain.Counterparty, error) {
// 	return domain.Counterparty{}, nil
// }
// func (m *mockCounterpartyRepo) Update(ctx context.Context, id int64, input domain.Counterparty) (domain.Counterparty, error) {
// 	return input, nil
// }
// func (m *mockCounterpartyRepo) SoftDelete(ctx context.Context, id int64) error {
// 	return nil
// }

// type mockABSClient struct {
// 	info *abs.ABSClientInfo
// 	err  error
// }

// func (m *mockABSClient) GetClientByINN(ctx context.Context, inn string) (*abs.ABSClientInfo, error) {
// 	return m.info, m.err
// }

// func TestCreateLegalEntity_RejectIndividual(t *testing.T) {
// 	repo := &mockCounterpartyRepo{}
// 	absMock := &mockABSClient{
// 		info: &abs.ABSClientInfo{
// 			INN:        "987654321",
// 			FullName:   "Рачабов Некрузчон Саймухторович",
// 			ClientType: "Физическое лицо",
// 			Phones:     []string{"992714475050"},
// 		},
// 	}
// 	svc := NewCounterpartyService(repo, absMock)

// 	_, err := svc.CreateLegalEntityFromABS(context.Background(), "test_user", "987654321", 1)
// 	if err == nil {
// 		t.Fatal("expected error when trying to create individual as legal entity, got nil")
// 	}
// 	expectedMsg := "является физическим лицом"
// 	if !contains(err.Error(), expectedMsg) {
// 		t.Errorf("expected error to contain %q, got %q", expectedMsg, err.Error())
// 	}
// }

// func TestCreateIndividual_RejectLegalEntity(t *testing.T) {
// 	repo := &mockCounterpartyRepo{}
// 	absMock := &mockABSClient{
// 		info: &abs.ABSClientInfo{
// 			INN:        "040008291",
// 			FullName:   `ҶДММ "АКТИВ БАНК"`,
// 			ClientType: "Юридическое лицо",
// 			Phones:     []string{"2230628"},
// 		},
// 	}
// 	svc := NewCounterpartyService(repo, absMock)

// 	_, err := svc.CreateIndividualFromABS(context.Background(), "test_user", "040008291", "ҶДММ Ромашка", 1)
// 	if err == nil {
// 		t.Fatal("expected error when trying to create legal entity as individual, got nil")
// 	}
// 	expectedMsg := "является юридическим лицом"
// 	if !contains(err.Error(), expectedMsg) {
// 		t.Errorf("expected error to contain %q, got %q", expectedMsg, err.Error())
// 	}
// }

// func TestCreateLegalEntity_Success(t *testing.T) {
// 	repo := &mockCounterpartyRepo{}
// 	absMock := &mockABSClient{
// 		info: &abs.ABSClientInfo{
// 			INN:        "040008291",
// 			FullName:   `ҶДММ "АКТИВ БАНК"`,
// 			ClientType: "Юридическое лицо",
// 			Phones:     []string{"2230628"},
// 		},
// 	}
// 	svc := NewCounterpartyService(repo, absMock)

// 	res, err := svc.CreateLegalEntityFromABS(context.Background(), "test_user", "040008291", 1)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if res.LLC != `ҶДММ "АКТИВ БАНК"` {
// 		t.Errorf("LLC: got %q, want %q", res.LLC, `ҶДММ "АКТИВ БАНК"`)
// 	}
// 	if res.Name != "" {
// 		t.Errorf("Name should be empty for legal entity, got %q", res.Name)
// 	}
// 	if res.ClientType != domain.ClientTypeLegalEntity {
// 		t.Errorf("ClientType: got %q, want %q", res.ClientType, domain.ClientTypeLegalEntity)
// 	}
// }

// func TestCreateIndividual_Success(t *testing.T) {
// 	repo := &mockCounterpartyRepo{}
// 	absMock := &mockABSClient{
// 		info: &abs.ABSClientInfo{
// 			INN:        "987654321",
// 			FullName:   "Рачабов Некрузчон Саймухторович",
// 			ClientType: "Физическое лицо",
// 			Phones:     []string{"992714475050"},
// 		},
// 	}
// 	svc := NewCounterpartyService(repo, absMock)

// 	res, err := svc.CreateIndividualFromABS(context.Background(), "test_user", "987654321", "ҶДММ Ромашка", 1)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if res.Name != "Рачабов Некрузчон Саймухторович" {
// 		t.Errorf("Name: got %q, want %q", res.Name, "Рачабов Некрузчон Саймухторович")
// 	}
// 	if res.LLC != "ҶДММ Ромашка" {
// 		t.Errorf("LLC: got %q, want %q", res.LLC, "ҶДММ Ромашка")
// 	}
// 	if res.ClientType != domain.ClientTypeIndividual {
// 		t.Errorf("ClientType: got %q, want %q", res.ClientType, domain.ClientTypeIndividual)
// 	}
// }

// func TestCreateSoleProprietor_Success(t *testing.T) {
// 	repo := &mockCounterpartyRepo{}
// 	absMock := &mockABSClient{
// 		info: &abs.ABSClientInfo{
// 			INN:        "987654321",
// 			FullName:   "Рачабов Некрузчон Саймухторович",
// 			ClientType: "Физическое лицо",
// 			Phones:     []string{"992714475050"},
// 		},
// 	}
// 	svc := NewCounterpartyService(repo, absMock)

// 	res, err := svc.CreateSoleProprietorFromABS(context.Background(), "test_user", "987654321", "ИП Рачабов", 1)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if res.Name != "Рачабов Некрузчон Саймухторович" {
// 		t.Errorf("Name: got %q, want %q", res.Name, "Рачабов Некрузчон Саймухторович")
// 	}
// 	if res.LLC != "ИП Рачабов" {
// 		t.Errorf("LLC: got %q, want %q", res.LLC, "ИП Рачабов")
// 	}
// 	if res.ClientType != domain.ClientTypeSoleProprietor {
// 		t.Errorf("ClientType: got %q, want %q", res.ClientType, domain.ClientTypeSoleProprietor)
// 	}
// }

// func TestCreateSoleProprietor_RejectLegalEntity(t *testing.T) {
// 	repo := &mockCounterpartyRepo{}
// 	absMock := &mockABSClient{
// 		info: &abs.ABSClientInfo{
// 			INN:        "040008291",
// 			FullName:   `ҶДММ "АКТИВ БАНК"`,
// 			ClientType: "Юридическое лицо",
// 			Phones:     []string{"2230628"},
// 		},
// 	}
// 	svc := NewCounterpartyService(repo, absMock)

// 	_, err := svc.CreateSoleProprietorFromABS(context.Background(), "test_user", "040008291", "ИП Актив", 1)
// 	if err == nil {
// 		t.Fatal("expected error when trying to create legal entity as sole proprietor, got nil")
// 	}
// 	expectedMsg := "является юридическим лицом"
// 	if !contains(err.Error(), expectedMsg) {
// 		t.Errorf("expected error to contain %q, got %q", expectedMsg, err.Error())
// 	}
// }

// func TestCreateSoleProprietor_WithABSSoleProprietorType(t *testing.T) {
// 	repo := &mockCounterpartyRepo{}
// 	absMock := &mockABSClient{
// 		info: &abs.ABSClientInfo{
// 			INN:        "045931660",
// 			FullName:   "БУРЯК ИВАН РАДИОНОВИЧ",
// 			ClientType: "Индивидуальный предприниматель",
// 			Phones:     []string{"992918616796"},
// 		},
// 	}
// 	svc := NewCounterpartyService(repo, absMock)

// 	res, err := svc.CreateSoleProprietorFromABS(context.Background(), "test_user", "045931660", "ИП Буряк", 1)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if res.Name != "БУРЯК ИВАН РАДИОНОВИЧ" {
// 		t.Errorf("Name: got %q, want %q", res.Name, "БУРЯК ИВАН РАДИОНОВИЧ")
// 	}
// 	if res.LLC != "ИП Буряк" {
// 		t.Errorf("LLC: got %q, want %q", res.LLC, "ИП Буряк")
// 	}
// 	if res.ClientType != domain.ClientTypeSoleProprietor {
// 		t.Errorf("ClientType: got %q, want %q", res.ClientType, domain.ClientTypeSoleProprietor)
// 	}
// }

// func TestCreateLegalEntity_RejectSoleProprietor(t *testing.T) {
// 	repo := &mockCounterpartyRepo{}
// 	absMock := &mockABSClient{
// 		info: &abs.ABSClientInfo{
// 			INN:        "045931660",
// 			FullName:   "БУРЯК ИВАН РАДИОНОВИЧ",
// 			ClientType: "Индивидуальный предприниматель",
// 			Phones:     []string{"992918616796"},
// 		},
// 	}
// 	svc := NewCounterpartyService(repo, absMock)

// 	_, err := svc.CreateLegalEntityFromABS(context.Background(), "test_user", "045931660", 1)
// 	if err == nil {
// 		t.Fatal("expected error when trying to create sole proprietor as legal entity, got nil")
// 	}
// 	expectedMsg := "является индивидуальным предпринимателем"
// 	if !contains(err.Error(), expectedMsg) {
// 		t.Errorf("expected error to contain %q, got %q", expectedMsg, err.Error())
// 	}
// }

// func TestCreateIndividual_RejectSoleProprietor(t *testing.T) {
// 	repo := &mockCounterpartyRepo{}
// 	absMock := &mockABSClient{
// 		info: &abs.ABSClientInfo{
// 			INN:        "045931660",
// 			FullName:   "БУРЯК ИВАН РАДИОНОВИЧ",
// 			ClientType: "Индивидуальный предприниматель",
// 			Phones:     []string{"992918616796"},
// 		},
// 	}
// 	svc := NewCounterpartyService(repo, absMock)

// 	_, err := svc.CreateIndividualFromABS(context.Background(), "test_user", "045931660", "ООО Тест", 1)
// 	if err == nil {
// 		t.Fatal("expected error when trying to create sole proprietor as individual, got nil")
// 	}
// 	expectedMsg := "является индивидуальным предпринимателем"
// 	if !contains(err.Error(), expectedMsg) {
// 		t.Errorf("expected error to contain %q, got %q", expectedMsg, err.Error())
// 	}
// }

// func TestCreateFromABS_SoleProprietor(t *testing.T) {
// 	repo := &mockCounterpartyRepo{}
// 	absMock := &mockABSClient{
// 		info: &abs.ABSClientInfo{
// 			INN:        "045931660",
// 			FullName:   "БУРЯК ИВАН РАДИОНОВИЧ",
// 			ClientType: "Индивидуальный предприниматель",
// 			Phones:     []string{"992918616796"},
// 		},
// 	}
// 	svc := NewCounterpartyService(repo, absMock)

// 	res, err := svc.CreateFromABS(context.Background(), "test_user", "ИП Буряк", "045931660", 1)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if res.Name != "БУРЯК ИВАН РАДИОНОВИЧ" {
// 		t.Errorf("Name: got %q, want %q", res.Name, "БУРЯК ИВАН РАДИОНОВИЧ")
// 	}
// 	if res.LLC != "ИП Буряк" {
// 		t.Errorf("LLC: got %q, want %q", res.LLC, "ИП Буряк")
// 	}
// 	if res.ClientType != domain.ClientTypeSoleProprietor {
// 		t.Errorf("ClientType: got %q, want %q", res.ClientType, domain.ClientTypeSoleProprietor)
// 	}
// }

// func contains(s, substr string) bool {
// 	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
// }
