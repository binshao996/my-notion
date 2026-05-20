package database

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bin-ke/my-notion/pkg/db"
	"gorm.io/gorm"
)

// --- config parsing structs ---

type filterNode struct {
	And        []filterCondition `json:"and"`
	Or         []filterCondition `json:"or"`
	PropertyID uint              `json:"property_id"`
	Operator   string            `json:"operator"`
	Value      any               `json:"value"`
}

type filterCondition struct {
	PropertyID uint   `json:"property_id"`
	Operator   string `json:"operator"`
	Value      any    `json:"value"`
}

type sortConfig struct {
	PropertyID uint   `json:"property_id"`
	Direction  string `json:"direction"`
}

type groupConfig struct {
	PropertyID uint `json:"property_id"`
}

type viewConfig struct {
	Filters          json.RawMessage `json:"filters"`
	Sorts            []sortConfig    `json:"sorts"`
	GroupBy          *groupConfig    `json:"groupBy"`
	HiddenProperties []uint          `json:"hidden_properties"`
}

// --- operator whitelist ---

var allowedOperators = map[string]bool{
	"equals": true, "not_equals": true,
	"contains": true, "not_contains": true,
	"is_empty": true, "is_not_empty": true,
	"greater_than": true, "less_than": true,
	"greater_than_or_equal": true, "less_than_or_equal": true,
	"starts_with": true, "ends_with": true,
}

// --- service ---

// QueryService translates view configuration (filters, sorts, groupBy) into
// parameterized GORM queries. All filter values use ? parameters — no raw SQL
// string building for user-supplied values.
type QueryService struct {
	DB *gorm.DB
}

// NewQueryService creates a QueryService backed by the given database.
func NewQueryService(database *gorm.DB) *QueryService {
	return &QueryService{DB: database}
}

// QueryRecords returns records for the given view with filters, sorts, and
// grouping applied.  It also returns the total number of matching records
// (before pagination).
func (s *QueryService) QueryRecords(databaseID uint, viewID uint, page, limit int) ([]db.Record, int64, error) {
	// 1. Load view
	var view db.View
	if err := s.DB.First(&view, viewID).Error; err != nil {
		return nil, 0, fmt.Errorf("view not found: %w", err)
	}

	// 2. Parse view config
	var vc viewConfig
	if err := json.Unmarshal([]byte(view.Config), &vc); err != nil {
		return nil, 0, fmt.Errorf("invalid view config: %w", err)
	}

	// 3. Load properties for the database so we know JSONB extraction paths
	var properties []db.Property
	if err := s.DB.Where("database_id = ?", databaseID).Find(&properties).Error; err != nil {
		return nil, 0, err
	}
	propMap := make(map[uint]db.Property, len(properties))
	for _, p := range properties {
		propMap[p.ID] = p
	}

	// 4. Count total matching records (filters only, no sort/group/pagination)
	countQuery := s.DB.Table("records").Where("records.database_id = ?", databaseID)
	var err error
	countQuery, err = applyFilters(countQuery, vc.Filters, propMap)
	if err != nil {
		return nil, 0, fmt.Errorf("applying filters for count: %w", err)
	}
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting records: %w", err)
	}

	// 5. Build the main query with filters, sorts, and grouping
	mainQuery := s.DB.Table("records").Joins("JOIN pages ON pages.id = records.page_id").Where("records.database_id = ?", databaseID)
	mainQuery, err = applyFilters(mainQuery, vc.Filters, propMap)
	if err != nil {
		return nil, 0, fmt.Errorf("applying filters: %w", err)
	}
	mainQuery, err = applySorts(mainQuery, vc.Sorts, propMap)
	if err != nil {
		return nil, 0, fmt.Errorf("applying sorts: %w", err)
	}
	mainQuery, err = applyGrouping(mainQuery, vc.GroupBy, propMap)
	if err != nil {
		return nil, 0, fmt.Errorf("applying grouping: %w", err)
	}

	// 6. Paginate
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	mainQuery = mainQuery.Offset((page - 1) * limit).Limit(limit)

	// 7. Fetch records
	var records []db.Record
	if err := mainQuery.Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("finding records: %w", err)
	}

	return records, total, nil
}

// --- filter helpers ---

// applyFilters parses the raw filters JSON and adds the appropriate JOINs and
// WHERE clauses to the query.  Returns the modified query.
func applyFilters(query *gorm.DB, filtersJSON json.RawMessage, propMap map[uint]db.Property) (*gorm.DB, error) {
	if len(filtersJSON) == 0 || string(filtersJSON) == "null" {
		return query, nil
	}

	var node filterNode
	if err := json.Unmarshal(filtersJSON, &node); err != nil {
		return query, err
	}

	// Determine filter type: compound (and/or) or single
	var conditions []filterCondition
	var isOr bool

	if len(node.And) > 0 {
		conditions = node.And
	} else if len(node.Or) > 0 {
		conditions = node.Or
		isOr = true
	} else if node.PropertyID > 0 && allowedOperators[node.Operator] {
		conditions = []filterCondition{{PropertyID: node.PropertyID, Operator: node.Operator, Value: node.Value}}
	} else {
		return query, nil
	}

	// Filter out conditions with unknown operators
	valid := make([]filterCondition, 0, len(conditions))
	for _, c := range conditions {
		if allowedOperators[c.Operator] {
			valid = append(valid, c)
		}
	}
	if len(valid) == 0 {
		return query, nil
	}

	// Join property_values once per unique property ID
	joined := make(map[uint]bool)
	for _, c := range valid {
		if joined[c.PropertyID] {
			continue
		}
		joined[c.PropertyID] = true

		if _, ok := propMap[c.PropertyID]; !ok {
			continue
		}

		alias := fmt.Sprintf("fv_%d", c.PropertyID)
		query = query.Joins(
			fmt.Sprintf("LEFT JOIN property_values %s ON records.id = %s.record_id AND %s.property_id = ?", alias, alias, alias),
			c.PropertyID,
		)
	}

	// Build WHERE clauses
	if isOr {
		var parts []string
		var args []any
		for _, c := range valid {
			sql, arg := buildFilterSQL(c, propMap)
			if sql != "" {
				parts = append(parts, sql)
				if arg != nil {
					args = append(args, arg)
				}
			}
		}
		if len(parts) > 0 {
			query = query.Where("("+strings.Join(parts, " OR ")+")", args...)
		}
	} else {
		for _, c := range valid {
			sql, arg := buildFilterSQL(c, propMap)
			if sql != "" {
				if arg != nil {
					query = query.Where(sql, arg)
				} else {
					query = query.Where(sql)
				}
			}
		}
	}

	return query, nil
}

// buildFilterSQL returns a parameterized WHERE fragment and its value for a
// single filter condition.  It returns ("", nil) if the property is unknown or
// the operator is unsupported.
func buildFilterSQL(c filterCondition, propMap map[uint]db.Property) (string, any) {
	prop, ok := propMap[c.PropertyID]
	if !ok {
		return "", nil
	}

	alias := fmt.Sprintf("fv_%d", c.PropertyID)
	path := getValuePath(alias, prop.Type)

	if prop.Type == "relation" {
		switch c.Operator {
		case "contains":
			return fmt.Sprintf("(%s)::jsonb @> to_jsonb(?)", path), c.Value
		case "not_contains":
			return fmt.Sprintf("NOT ((%s)::jsonb @> to_jsonb(?))", path), c.Value
		case "is_empty":
			return fmt.Sprintf("(%s IS NULL OR jsonb_array_length(%s) = 0)", path, path), nil
		case "is_not_empty":
			return fmt.Sprintf("(%s IS NOT NULL AND jsonb_array_length(%s) > 0)", path, path), nil
		}
	}

	switch c.Operator {
	case "equals":
		return fmt.Sprintf("%s = ?", path), c.Value
	case "not_equals":
		return fmt.Sprintf("%s IS DISTINCT FROM ?", path), c.Value
	case "contains":
		return fmt.Sprintf("%s LIKE ?", path), fmt.Sprintf("%%%v%%", c.Value)
	case "not_contains":
		return fmt.Sprintf("%s NOT LIKE ?", path), fmt.Sprintf("%%%v%%", c.Value)
	case "is_empty":
		return fmt.Sprintf("(%s IS NULL OR %s = '')", path, path), nil
	case "is_not_empty":
		return fmt.Sprintf("(%s IS NOT NULL AND %s != '')", path, path), nil
	case "greater_than":
		return fmt.Sprintf("%s > ?", path), c.Value
	case "less_than":
		return fmt.Sprintf("%s < ?", path), c.Value
	case "greater_than_or_equal":
		return fmt.Sprintf("%s >= ?", path), c.Value
	case "less_than_or_equal":
		return fmt.Sprintf("%s <= ?", path), c.Value
	case "starts_with":
		return fmt.Sprintf("%s LIKE ?", path), fmt.Sprintf("%v%%", c.Value)
	case "ends_with":
		return fmt.Sprintf("%s LIKE ?", path), fmt.Sprintf("%%%v", c.Value)
	default:
		return "", nil
	}
}

// --- sort helpers ---

// applySorts adds LEFT JOINs and ORDER BY clauses for each sort config.
func applySorts(query *gorm.DB, sorts []sortConfig, propMap map[uint]db.Property) (*gorm.DB, error) {
	for _, s := range sorts {
		prop, ok := propMap[s.PropertyID]
		if !ok {
			continue
		}

		alias := fmt.Sprintf("sv_%d", s.PropertyID)
		query = query.Joins(
			fmt.Sprintf("LEFT JOIN property_values %s ON records.id = %s.record_id AND %s.property_id = ?", alias, alias, alias),
			s.PropertyID,
		)

		path := getValuePath(alias, prop.Type)
		dir := "ASC"
		if strings.ToLower(s.Direction) == "desc" {
			dir = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", path, dir))
	}
	return query, nil
}

// --- group helpers ---

// applyGrouping adds a LEFT JOIN and GROUP BY for the groupBy config.
func applyGrouping(query *gorm.DB, groupBy *groupConfig, propMap map[uint]db.Property) (*gorm.DB, error) {
	if groupBy == nil {
		return query, nil
	}

	prop, ok := propMap[groupBy.PropertyID]
	if !ok {
		return query, nil
	}

	alias := fmt.Sprintf("gv_%d", groupBy.PropertyID)
	query = query.Joins(
		fmt.Sprintf("LEFT JOIN property_values %s ON records.id = %s.record_id AND %s.property_id = ?", alias, alias, alias),
		groupBy.PropertyID,
	)

	path := getValuePath(alias, prop.Type)
	query = query.Select(fmt.Sprintf("records.*, COUNT(*) AS group_count")).
		Group("records.id, " + path)

	return query, nil
}

// --- shared helpers ---

// getValuePath returns the SQL expression to extract the comparable value from
// a property_value's JSONB value column based on the property type.
//
//	alias is the table alias of the property_values join (e.g. "fv_1").
func getValuePath(alias string, propType string) string {
	switch propType {
	case "number":
		return fmt.Sprintf("(%s.value ->> 'number')::numeric", alias)
	case "checkbox":
		return fmt.Sprintf("(%s.value ->> 'checked')::boolean", alias)
	case "select", "status":
		return fmt.Sprintf("%s.value ->> 'select'", alias)
	case "multi_select":
		return fmt.Sprintf("%s.value ->> 'multi_select'", alias)
	case "date":
		return fmt.Sprintf("%s.value ->> 'date'", alias)
	case "relation":
		return fmt.Sprintf("%s.value -> 'relation'", alias)
	case "created_time":
		return "pages.created_at"
	case "last_edited_time":
		return "pages.updated_at"
	default: // title, text, url, email, phone, person, files
		return fmt.Sprintf("%s.value ->> 'text'", alias)
	}
}
