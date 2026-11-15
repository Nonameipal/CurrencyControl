package response

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// DiffTracker аккумулирует изменения полей при редактировании сущности для журнала аудита.
type DiffTracker struct {
	changes []string
}

// NewDiffTracker создает новый экземпляр трекера изменений.
func NewDiffTracker() *DiffTracker {
	return &DiffTracker{changes: make([]string, 0, 8)}
}

// TrackString фиксирует изменение строкового поля.
func (d *DiffTracker) TrackString(fieldName, oldVal, newVal string) {
	newVal = strings.TrimSpace(newVal)
	oldVal = strings.TrimSpace(oldVal)
	if newVal != "" && newVal != oldVal {
		d.changes = append(d.changes, fmt.Sprintf("%s: '%s' -> '%s'", fieldName, oldVal, newVal))
	}
}

// TrackStringPtr фиксирует изменение указателя на строку.
func (d *DiffTracker) TrackStringPtr(fieldName string, oldVal *string, newVal *string) {
	oldStr := ""
	if oldVal != nil {
		oldStr = strings.TrimSpace(*oldVal)
	}
	newStr := ""
	if newVal != nil {
		newStr = strings.TrimSpace(*newVal)
	}
	if newStr != "" && newStr != oldStr {
		d.changes = append(d.changes, fmt.Sprintf("%s: '%s' -> '%s'", fieldName, oldStr, newStr))
	}
}

// TrackFloat фиксирует изменение числа с плавающей точкой.
func (d *DiffTracker) TrackFloat(fieldName string, oldVal, newVal float64) {
	if newVal > 0 && math.Abs(oldVal-newVal) > 0.0001 {
		d.changes = append(d.changes, fmt.Sprintf("%s: '%.2f' -> '%.2f'", fieldName, oldVal, newVal))
	}
}

// TrackInt фиксирует изменение целочисленного поля.
func (d *DiffTracker) TrackInt(fieldName string, oldVal, newVal int) {
	if newVal != oldVal {
		d.changes = append(d.changes, fmt.Sprintf("%s: '%d' -> '%d'", fieldName, oldVal, newVal))
	}
}

// TrackIntPtr фиксирует изменение указателя на целое число.
func (d *DiffTracker) TrackIntPtr(fieldName string, oldVal *int, newVal *int) {
	if newVal == nil && oldVal == nil {
		return
	}
	oldStr := "нет"
	if oldVal != nil {
		oldStr = fmt.Sprintf("%d", *oldVal)
	}
	newStr := "нет"
	if newVal != nil {
		newStr = fmt.Sprintf("%d", *newVal)
	}
	if oldStr != newStr {
		d.changes = append(d.changes, fmt.Sprintf("%s: '%s' -> '%s'", fieldName, oldStr, newStr))
	}
}

// TrackDate фиксирует изменение даты.
func (d *DiffTracker) TrackDate(fieldName string, oldVal time.Time, newVal *time.Time) {
	if newVal != nil && !newVal.IsZero() {
		oldStr := oldVal.Format("02.01.2006")
		newStr := newVal.Format("02.01.2006")
		if oldStr != newStr {
			d.changes = append(d.changes, fmt.Sprintf("%s: '%s' -> '%s'", fieldName, oldStr, newStr))
		}
	}
}

// TrackDatePtr фиксирует изменение указателя на дату.
func (d *DiffTracker) TrackDatePtr(fieldName string, oldVal *time.Time, newVal *time.Time) {
	if newVal == nil && oldVal == nil {
		return
	}
	oldStr := "нет"
	if oldVal != nil && !oldVal.IsZero() {
		oldStr = oldVal.Format("02.01.2006")
	}
	newStr := "нет"
	if newVal != nil && !newVal.IsZero() {
		newStr = newVal.Format("02.01.2006")
	}
	if oldStr != newStr {
		d.changes = append(d.changes, fmt.Sprintf("%s: '%s' -> '%s'", fieldName, oldStr, newStr))
	}
}

// TrackBool фиксирует изменение булева поля.
func (d *DiffTracker) TrackBool(fieldName string, oldVal, newVal bool) {
	if oldVal != newVal {
		d.changes = append(d.changes, fmt.Sprintf("%s: '%t' -> '%t'", fieldName, oldVal, newVal))
	}
}

// TrackCustom добавляет произвольное описание изменения (например, о прикреплении файла).
func (d *DiffTracker) TrackCustom(desc string) {
	if strings.TrimSpace(desc) != "" {
		d.changes = append(d.changes, desc)
	}
}

// HasChanges возвращает true, если зафиксировано хотя бы одно изменение.
func (d *DiffTracker) HasChanges() bool {
	return len(d.changes) > 0
}

// String возвращает строковое представление всех изменений через точку с запятой.
func (d *DiffTracker) String() string {
	if len(d.changes) == 0 {
		return "без изменений"
	}
	return strings.Join(d.changes, "; ")
}
