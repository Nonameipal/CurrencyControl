package http

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DictionaryHandler struct {
	db *pgxpool.Pool
}

func NewDictionaryHandler(db *pgxpool.Pool) *DictionaryHandler {
	return &DictionaryHandler{db: db}
}

type Country struct {
	ID     int    `json:"id"`
	NameRu string `json:"name_ru"`
}

// @Summary Поиск стран
// @Description Поиск по справочнику стран по части названия без учета регистра.
// @Tags Dictionary
// @Produce json
// @Param q query string false "Строка для поиска"
// @Success 200 {array} Country
// @Router /api/countries [get]
func (h *DictionaryHandler) SearchCountries(w http.ResponseWriter, r *http.Request) {
	queryParam := strings.TrimSpace(r.URL.Query().Get("q"))

	countries := []Country{}

	query := "SELECT id, name_ru FROM countries"
	var err error
	var rows pgx.Rows

	if queryParam != "" {
		query += " WHERE name_ru ILIKE $1 ORDER BY name_ru"
		rows, err = h.db.Query(r.Context(), query, "%"+queryParam+"%")
	} else {
		query += " ORDER BY name_ru"
		rows, err = h.db.Query(r.Context(), query)
	}

	if err != nil {
		handleError(w, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var c Country
		if err := rows.Scan(&c.ID, &c.NameRu); err != nil {
			handleError(w, err)
			return
		}
		countries = append(countries, c)
	}

	writeJSON(w, http.StatusOK, countries)
}

type Branch struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// @Summary Получение списка филиалов
// @Description Возвращает список всех филиалов для экрана выбора.
// @Tags Branches
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Success 200 {array} Branch
// @Router /api/branches [get]
func (h *DictionaryHandler) GetBranches(w http.ResponseWriter, r *http.Request) {
	branches := []Branch{}

	query := "SELECT id, name FROM branches ORDER BY id"
	rows, err := h.db.Query(r.Context(), query)
	if err != nil {
		handleError(w, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var b Branch
		if err := rows.Scan(&b.ID, &b.Name); err != nil {
			handleError(w, err)
			return
		}
		branches = append(branches, b)
	}

	writeJSON(w, http.StatusOK, branches)
}

type Currency struct {
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
// @Success 200 {array} Currency
// @Router /api/currencies [get]
func (h *DictionaryHandler) SearchCurrencies(w http.ResponseWriter, r *http.Request) {
	queryParam := r.URL.Query().Get("q")

	currencies := []Currency{}

	query := "SELECT id, code, numeric_code, name_ru FROM currencies"
	var err error
	var rows pgx.Rows

	if queryParam != "" {
		query += " WHERE name_ru ILIKE $1 OR code ILIKE $1 ORDER BY code"
		rows, err = h.db.Query(r.Context(), query, "%"+queryParam+"%")
	} else {
		query += " ORDER BY code"
		rows, err = h.db.Query(r.Context(), query)
	}

	if err != nil {
		handleError(w, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var c Currency
		if err := rows.Scan(&c.ID, &c.Code, &c.NumericCode, &c.NameRu); err != nil {
			handleError(w, err)
			return
		}
		currencies = append(currencies, c)
	}

	writeJSON(w, http.StatusOK, currencies)
}
