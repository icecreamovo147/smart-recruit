// Package tenantgorm provides a fail-closed GORM boundary for tenant-owned
// tables. It complements explicit domain authorization and protects ordinary
// ORM reads/writes from accidentally crossing an enterprise boundary.
package tenantgorm

import (
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	platformmetadata "smart-recruit-platform-go/metadata"
)

var ErrMissingTenantContext = errors.New("staff database operation is missing tenant context")

type Plugin struct {
	tables map[string]struct{}
	mixed  map[string]struct{}
}

func New(tables ...string) *Plugin {
	set := make(map[string]struct{}, len(tables))
	for _, table := range tables {
		if table = strings.TrimSpace(table); table != "" {
			set[table] = struct{}{}
		}
	}
	return &Plugin{tables: set, mixed: map[string]struct{}{}}
}

// NewWithMixed registers strict tenant-owned tables and mixed-scope tables.
// Staff reads of mixed tables see their tenant overrides plus global defaults;
// staff mutations can affect tenant rows only.
func NewWithMixed(owned, mixed []string) *Plugin {
	p := New(owned...)
	for _, table := range mixed {
		if table = strings.TrimSpace(table); table != "" {
			p.tables[table] = struct{}{}
			p.mixed[table] = struct{}{}
		}
	}
	return p
}

func (p *Plugin) Name() string { return "smart_recruit:tenant_boundary" }

func (p *Plugin) Initialize(db *gorm.DB) error {
	if err := db.Callback().Create().Before("gorm:create").Register(p.Name()+":create", p.beforeCreate); err != nil {
		return err
	}
	if err := db.Callback().Query().Before("gorm:query").Register(p.Name()+":query", p.scopeQuery); err != nil {
		return err
	}
	if err := db.Callback().Row().Before("gorm:row").Register(p.Name()+":row", p.scopeQuery); err != nil {
		return err
	}
	if err := db.Callback().Update().Before("gorm:update").Register(p.Name()+":update", p.beforeUpdate); err != nil {
		return err
	}
	return db.Callback().Delete().Before("gorm:delete").Register(p.Name()+":delete", p.scopeMutation)
}

func (p *Plugin) beforeUpdate(db *gorm.DB) {
	if _, _, ok := p.table(db.Statement.Table); !ok {
		return
	}
	tenantID, apply := p.tenant(db)
	if db.Error != nil || !apply {
		return
	}
	p.scopeMutation(db)
	db.Statement.SetColumn("tenant_id", tenantID)
}

func (p *Plugin) beforeCreate(db *gorm.DB) {
	if _, _, ok := p.table(db.Statement.Table); !ok {
		return
	}
	tenantID, apply := p.tenant(db)
	if db.Error != nil || !apply {
		return
	}
	db.Statement.SetColumn("tenant_id", tenantID)
}

func (p *Plugin) scopeQuery(db *gorm.DB) {
	qualifier, mixed, ok := p.table(db.Statement.Table)
	if !ok {
		return
	}
	tenantID, apply := p.tenant(db)
	if db.Error != nil || !apply {
		return
	}
	column := clause.Column{Table: qualifier, Name: "tenant_id"}
	expression := clause.Expression(clause.Eq{Column: column, Value: tenantID})
	if mixed {
		expression = clause.Or(clause.Eq{Column: column, Value: tenantID}, clause.Eq{Column: column, Value: nil})
	}
	db.Statement.AddClause(clause.Where{Exprs: []clause.Expression{expression}})
}

func (p *Plugin) scopeMutation(db *gorm.DB) {
	qualifier, _, ok := p.table(db.Statement.Table)
	if !ok {
		return
	}
	tenantID, apply := p.tenant(db)
	if db.Error != nil || !apply {
		return
	}
	db.Statement.AddClause(clause.Where{Exprs: []clause.Expression{clause.Eq{Column: clause.Column{Table: qualifier, Name: "tenant_id"}, Value: tenantID}}})
}

func (p *Plugin) tenant(db *gorm.DB) (int64, bool) {
	ctx := db.Statement.Context
	tenantID := platformmetadata.GetAuthTenantID(ctx)
	if tenantID > 0 {
		return tenantID, true
	}
	if platformmetadata.GetAuthAccountType(ctx) == "staff" {
		db.AddError(ErrMissingTenantContext)
	}
	return 0, false
}

func (p *Plugin) table(statementTable string) (string, bool, bool) {
	parts := strings.Fields(strings.ReplaceAll(statementTable, "`", ""))
	if len(parts) == 0 {
		return "", false, false
	}
	table := parts[0]
	if _, ok := p.tables[table]; !ok {
		return "", false, false
	}
	qualifier := table
	if len(parts) >= 2 {
		if strings.EqualFold(parts[1], "AS") && len(parts) >= 3 {
			qualifier = parts[2]
		} else {
			qualifier = parts[1]
		}
	}
	_, isMixed := p.mixed[table]
	return qualifier, isMixed, true
}
