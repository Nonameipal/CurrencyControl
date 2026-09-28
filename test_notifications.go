package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"CurrencyControl/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost user=postgres password=Noname0212 dbname=currency_control port=5432 sslmode=disable TimeZone=Asia/Dushanbe"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	repo := repository.NewContractRepository(db)
	
	notifications, _ := repo.GetExpiringContracts(context.Background(), 5100)
	fmt.Printf("Found %d notifications for branch 14\n", len(notifications))
	for _, n := range notifications {
		fmt.Printf("- %s: %s (days left: %d)\n", n.Type, n.Title, n.DaysLeft)
	}

	type temp struct {
		ID int64
		ContractNumber string
		ContractEndDate *time.Time
		BranchID int
	}
	var temps []temp
	db.Raw("SELECT c.id, c.contract_number, c.contract_end_date, comp.branch_id FROM contracts c JOIN counterparties comp ON c.client_id = comp.id WHERE c.deleted_at IS NULL").Scan(&temps)
	for _, c := range temps {
		endStr := "nil"
		if c.ContractEndDate != nil {
			endStr = c.ContractEndDate.Format("2006-01-02")
		}
		fmt.Printf("- ID: %d, Num: %s, EndDate: %s, BranchID: %d\n", c.ID, c.ContractNumber, endStr, c.BranchID)
	}
}
