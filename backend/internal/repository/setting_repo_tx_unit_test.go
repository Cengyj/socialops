package repository

import (
	"context"
	"database/sql"
	"net/url"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/Wei-Shaw/socialops/ent/setting"
	"github.com/Wei-Shaw/socialops/internal/service"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newSettingRepoSQLite(t *testing.T) (*settingRepository, *dbent.Client) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+url.QueryEscape(t.Name())+"?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable sqlite foreign keys: %v", err)
	}

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return NewSettingRepository(client).(*settingRepository), client
}

func TestSettingRepositorySetUsesContextTransaction(t *testing.T) {
	ctx := context.Background()
	repo, client := newSettingRepoSQLite(t)

	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	txCtx := dbent.NewTxContext(ctx, tx)

	if err := repo.Set(txCtx, "tx_setting_set", "value"); err != nil {
		_ = tx.Rollback()
		t.Fatalf("set setting: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback tx: %v", err)
	}

	count, err := client.Setting.Query().Where(setting.KeyEQ("tx_setting_set")).Count(ctx)
	if err != nil {
		t.Fatalf("count settings: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected Set to participate in context transaction rollback, got %d persisted setting(s)", count)
	}
}

func TestSettingRepositorySetMultipleUsesContextTransaction(t *testing.T) {
	ctx := context.Background()
	repo, client := newSettingRepoSQLite(t)

	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	txCtx := dbent.NewTxContext(ctx, tx)

	if err := repo.SetMultiple(txCtx, map[string]string{
		"tx_setting_multi_one": "one",
		"tx_setting_multi_two": "two",
	}); err != nil {
		_ = tx.Rollback()
		t.Fatalf("set multiple settings: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback tx: %v", err)
	}

	count, err := client.Setting.Query().
		Where(setting.KeyIn("tx_setting_multi_one", "tx_setting_multi_two")).
		Count(ctx)
	if err != nil {
		t.Fatalf("count settings: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected SetMultiple to participate in context transaction rollback, got %d persisted setting(s)", count)
	}
}

func TestSettingRepositoryDeleteUsesContextTransaction(t *testing.T) {
	ctx := context.Background()
	repo, client := newSettingRepoSQLite(t)

	if err := repo.Set(ctx, "tx_setting_delete", "keep"); err != nil {
		t.Fatalf("seed setting: %v", err)
	}

	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	txCtx := dbent.NewTxContext(ctx, tx)

	if err := repo.Delete(txCtx, "tx_setting_delete"); err != nil {
		_ = tx.Rollback()
		t.Fatalf("delete setting: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback tx: %v", err)
	}

	value, err := repo.GetValue(ctx, "tx_setting_delete")
	if err != nil {
		t.Fatalf("get setting after rolled back delete: %v", err)
	}
	if value != "keep" {
		t.Fatalf("expected rolled back delete to preserve value %q, got %q", "keep", value)
	}
}

func TestSettingRepositoryReadsUseContextTransaction(t *testing.T) {
	ctx := context.Background()
	repo, client := newSettingRepoSQLite(t)

	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	txCtx := dbent.NewTxContext(ctx, tx)
	t.Cleanup(func() { _ = tx.Rollback() })

	_, err = tx.Client().Setting.Create().
		SetKey("tx_setting_read").
		SetValue("uncommitted").
		Save(txCtx)
	if err != nil {
		t.Fatalf("seed transactional setting: %v", err)
	}

	value, err := repo.GetValue(txCtx, "tx_setting_read")
	if err != nil {
		t.Fatalf("get value in transaction: %v", err)
	}
	if value != "uncommitted" {
		t.Fatalf("expected transactional read value %q, got %q", "uncommitted", value)
	}

	values, err := repo.GetMultiple(txCtx, []string{"tx_setting_read"})
	if err != nil {
		t.Fatalf("get multiple in transaction: %v", err)
	}
	if values["tx_setting_read"] != "uncommitted" {
		t.Fatalf("expected transactional GetMultiple value %q, got %q", "uncommitted", values["tx_setting_read"])
	}

	all, err := repo.GetAll(txCtx)
	if err != nil {
		t.Fatalf("get all in transaction: %v", err)
	}
	if all["tx_setting_read"] != "uncommitted" {
		t.Fatalf("expected transactional GetAll value %q, got %q", "uncommitted", all["tx_setting_read"])
	}

	got, err := repo.Get(txCtx, "tx_setting_read")
	if err != nil {
		t.Fatalf("get setting in transaction: %v", err)
	}
	if got.Value != "uncommitted" {
		t.Fatalf("expected transactional Get value %q, got %q", "uncommitted", got.Value)
	}
}

func TestSettingRepositoryReadsUseContextTransactionNotFound(t *testing.T) {
	ctx := context.Background()
	repo, client := newSettingRepoSQLite(t)

	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	txCtx := dbent.NewTxContext(ctx, tx)
	t.Cleanup(func() { _ = tx.Rollback() })

	_, err = repo.Get(txCtx, "tx_setting_missing")
	if err == nil {
		t.Fatalf("expected missing setting error")
	}
	if err != service.ErrSettingNotFound {
		t.Fatalf("expected ErrSettingNotFound, got %v", err)
	}
}
