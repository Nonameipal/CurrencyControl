package http

import (
	"net/http"
	"strings"

	"gorm.io/gorm"
)

type DictionaryHandler struct {
	db *gorm.DB
}
func NewDictionaryHandler(db *gorm.DB) *DictionaryHandler {
	return &DictionaryHandler{db: db}
}
type CountryItem struct {
	ID     int    `json:"id"`
	NameRu string `json:"name_ru"`
}

// @Summary Поиск стран
// @Description Поиск по справочнику стран по части названия без учета регистра.
// @Tags Dictionary
// @Produce json
// @Param q query string false "Строка для поиска"
// @Success 200 {array} CountryItem
// @Router /api/countries [get]
func (h *DictionaryHandler) SearchCountries(w http.ResponseWriter, r *http.Request) {
	queryParam := strings.TrimSpace(r.URL.Query().Get("q"))
	var countries []CountryItem
	q := h.db.WithContext(r.Context()).Table("countries").
		Select("id, name_ru").
		Order("name_ru ASC")
	if queryParam != "" {
		q = q.Where("name_ru ILIKE ?", "%"+queryParam+"%")
	}
	if err := q.Scan(&countries).Error; err != nil {
		handleError(w, err)
		return
	}
	if countries == nil {
		countries = []CountryItem{}
	}
	writeJSON(w, http.StatusOK, countries)
}
type BranchItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// @Summary Получение списка филиалов
// @Description Возвращает список всех филиалов для экрана выбора.
// @Tags Branches
// @Produce json
// @Success 200 {array} BranchItem
// @Router /api/branches [get]
func (h *DictionaryHandler) GetBranches(w http.ResponseWriter, r *http.Request) {
	var branches []BranchItem
	if err := h.db.WithContext(r.Context()).Table("branches").
		Select("id, name").
		Where("deleted_at IS NULL").
		Order("id ASC").
		Scan(&branches).Error; err != nil {
		handleError(w, err)
		return
	}
	if branches == nil {
		branches = []BranchItem{}
	}
	writeJSON(w, http.StatusOK, branches)
}
type CurrencyItem struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	NumericCode int    `json:"numeric_code"`
	NameRu      string `json:"name_ru"`
}

// @Summary Поиск валют
// @Description Поиск по справочнику валют.
// @Tags Dictionary
// @Produce json
// @Param q query string false "Строка для поиска (по названию или коду)"
// @Success 200 {array} CurrencyItem
// @Router /api/currencies [get]
func (h *DictionaryHandler) SearchCurrencies(w http.ResponseWriter, r *http.Request) {
	queryParam := strings.TrimSpace(r.URL.Query().Get("q"))
	var currencies []CurrencyItem
	q := h.db.WithContext(r.Context()).Table("currencies").
		Select("id, code, numeric_code, name_ru").
		Order("code ASC")
	if queryParam != "" {
		q = q.Where("name_ru ILIKE ? OR code ILIKE ?", "%"+queryParam+"%", "%"+queryParam+"%")
	}
	if err := q.Scan(&currencies).Error; err != nil {
		handleError(w, err)
		return
	}
	if currencies == nil {
		currencies = []CurrencyItem{}
	}
	writeJSON(w, http.StatusOK, currencies)
}
