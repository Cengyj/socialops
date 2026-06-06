package repository

import (
	"context"
	"database/sql"
	"net/url"
	"testing"

	dbent "github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/ent/enttest"
	"github.com/Wei-Shaw/socialops/ent/redeemcode"
	"github.com/Wei-Shaw/socialops/internal/service"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newRedeemCodeRepoSQLite(t *testing.T) (*redeemCodeRepository, *dbent.Client) {
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

	return NewRedeemCodeRepository(client).(*redeemCodeRepository), client
}

func TestRedeemCodeRepositoryCreateUsesContextTransaction(t *testing.T) {
	ctx := context.Background()
	repo, client := newRedeemCodeRepoSQLite(t)

	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	txCtx := dbent.NewTxContext(ctx, tx)

	code := &service.RedeemCode{
		Code:   "TX-CREATE-ROLLBACK",
		Type:   service.RedeemTypeBalance,
		Value:  10,
		Status: service.StatusUnused,
	}
	if err := repo.Create(txCtx, code); err != nil {
		_ = tx.Rollback()
		t.Fatalf("create redeem code: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback tx: %v", err)
	}

	count, err := client.RedeemCode.Query().
		Where(redeemcode.CodeEQ("TX-CREATE-ROLLBACK")).
		Count(ctx)
	if err != nil {
		t.Fatalf("count redeem codes: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected Create to participate in context transaction rollback, got %d persisted code(s)", count)
	}
}

func TestRedeemCodeRepositoryCreateBatchUsesContextTransaction(t *testing.T) {
	ctx := context.Background()
	repo, client := newRedeemCodeRepoSQLite(t)

	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	txCtx := dbent.NewTxContext(ctx, tx)

	codes := []service.RedeemCode{
		{Code: "TX-BATCH-ROLLBACK-1", Type: service.RedeemTypeBalance, Value: 10, Status: service.StatusUnused},
		{Code: "TX-BATCH-ROLLBACK-2", Type: service.RedeemTypeBalance, Value: 20, Status: service.StatusUnused},
	}
	if err := repo.CreateBatch(txCtx, codes); err != nil {
		_ = tx.Rollback()
		t.Fatalf("create redeem code batch: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback tx: %v", err)
	}

	count, err := client.RedeemCode.Query().
		Where(redeemcode.CodeIn("TX-BATCH-ROLLBACK-1", "TX-BATCH-ROLLBACK-2")).
		Count(ctx)
	if err != nil {
		t.Fatalf("count redeem codes: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected CreateBatch to participate in context transaction rollback, got %d persisted code(s)", count)
	}
}
