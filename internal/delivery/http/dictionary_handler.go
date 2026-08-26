package http

import (
	"net/http"

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
// @Description Поиск по справочнику стран. Можно использовать % для поиска по части слова (например, %еспублик%).
// @Tags Dictionary
// @Accept multipart/form-data
// @Produce json
// @Param q formData string false "Строка для поиска"
// @Success 200 {array} Country
// @Router /api/countries [post]
func (h *DictionaryHandler) SearchCountries(w http.ResponseWriter, r *http.Request) {
	queryParam := r.FormValue("q")
	
	countries := []Country{}
	
	query := "SELECT id, name_ru FROM countries"
	var err error
	var rows pgx.Rows

	if queryParam != "" {
		query += " WHERE name_ru ILIKE  ORDER BY name_ru LIMIT 50"
		rows, err = h.db.Query(r.Context(), query, queryParam)
	} else {
		query += " ORDER BY name_ru LIMIT 50"
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