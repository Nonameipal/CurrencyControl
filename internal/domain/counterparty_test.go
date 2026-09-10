package domain_test

import (
	"testing"

	"CurrencyControl/internal/domain"
)

func TestCounterpartyPhonesAndAccounts(t *testing.T) {
	c := domain.Counterparty{}

	phones := []string{"+992900000001", "+992900000002"}
	c.SetPhones(phones)

	gotPhones := c.GetPhones()
	if len(gotPhones) != 2 || gotPhones[0] != phones[0] || gotPhones[1] != phones[1] {
		t.Fatalf("expected %v, got %v", phones, gotPhones)
	}

	accounts := []string{"20202972000000000001", "20202972000000000002"}
	c.SetAccounts(accounts)

	gotAccounts := c.GetAccounts()
	if len(gotAccounts) != 2 || gotAccounts[0] != accounts[0] || gotAccounts[1] != accounts[1] {
		t.Fatalf("expected %v, got %v", accounts, gotAccounts)
	}
}
