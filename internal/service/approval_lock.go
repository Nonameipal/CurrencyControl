package service

import (
	"fmt"

	"CurrencyControl/internal/domain"
)

func validateApprovalStatusForUpdate(status, createdBy, updatedBy string) (bool, error) {
	switch status {
	case domain.ApprovalStatusPendingCurrencyControl:
		return false, fmt.Errorf("документ на рассмотрении валютного контроля — редактирование запрещено")
	case domain.ApprovalStatusPendingCompliance:
		return false, fmt.Errorf("документ на рассмотрении комплаенса — редактирование запрещено")
	case domain.ApprovalStatusRevisionRequired:
		if updatedBy != createdBy {
			return false, fmt.Errorf("редактировать документ на доработке может только его создатель")
		}
		return true, nil
	case domain.ApprovalStatusApproved:
		return false, nil
	case domain.ApprovalStatusRejectedCurrencyControl, domain.ApprovalStatusRejectedCompliance:
		return false, fmt.Errorf("документ отклонён и перемещён в корзину — редактирование невозможно")
	default:
		return false, nil
	}
}


func validateApprovalStatusForDelete(status string) error {
	switch status {
	case domain.ApprovalStatusPendingCurrencyControl:
		return fmt.Errorf("документ на рассмотрении валютного контроля — удаление запрещено")
	case domain.ApprovalStatusPendingCompliance:
		return fmt.Errorf("документ на рассмотрении комплаенс-контроля — удаление запрещено")
	case domain.ApprovalStatusRevisionRequired:
		return fmt.Errorf("нельзя удалить документ, отправленный на доработку")
	case domain.ApprovalStatusRejectedCurrencyControl, domain.ApprovalStatusRejectedCompliance:
		return fmt.Errorf("документ уже отклонён и находится в корзине")
	default:
		return nil
	}
}
