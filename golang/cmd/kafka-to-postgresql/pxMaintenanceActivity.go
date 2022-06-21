package main

/*
	Warning:
		This file is based on old source code, not on documentation !
*/

import (
	"context"
	"database/sql"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/lib/pq"
	"github.com/united-manufacturing-hub/united-manufacturing-hub/internal"
	"go.uber.org/zap"
	"time"
)

type AddMaintenanceActivity struct{}

type addMaintenanceActivity struct {
	TimestampMs   *uint64 `json:"timestamp_ms"`
	ComponentName *string `json:"component"`
	Activity      *int32  `json:"activity"`
}

// ProcessMessages processes a AddMaintenanceActivity kafka message, by creating an database connection, decoding the json payload, retrieving the required additional database id's (like AssetTableID or ProductTableID) and then inserting it into the database and commiting
func (c AddMaintenanceActivity) ProcessMessages(msg internal.ParsedMessage) (putback bool, err error, forcePbTopic bool) {

	txnCtx, txnCtxCl := context.WithDeadline(context.Background(), time.Now().Add(internal.FiveSeconds))
	// txnCtxCl is the cancel function of the context, used in the transaction creation.
	// It is deferred to automatically release the allocated resources, once the function returns
	defer txnCtxCl()
	var txn *sql.Tx = nil
	txn, err = db.BeginTx(txnCtx, nil)
	if err != nil {
		zap.S().Errorf("Error starting transaction: %s", err.Error())
		return true, err, false
	}

	isCommited := false
	defer func() {
		if !isCommited && !isDryRun {
			err = txn.Rollback()
			if err != nil {
				zap.S().Errorf("Error rolling back transaction: %s", err.Error())
			} else {
				zap.S().Warnf("Rolled back transaction !")
			}
		}
	}()

	// sC is the payload, parsed as addMaintenanceActivity
	var sC addMaintenanceActivity
	err = jsoniter.Unmarshal(msg.Payload, &sC)
	if err != nil {
		zap.S().Warnf("Failed to unmarshal message: %s", err.Error())
		return false, err, true
	}
	if !internal.IsValidStruct(sC, []string{}) {
		zap.S().Warnf("Invalid message: %s, inserting into putback !", string(msg.Payload))
		return true, nil, true
	}

	AssetTableID, success := GetAssetTableID(msg.CustomerId, msg.Location, msg.AssetId)
	if !success {
		zap.S().Warnf("Failed to get AssetTableID")
		return true, fmt.Errorf("failed to get AssetTableID for CustomerId: %s, Location: %s, AssetId: %s", msg.CustomerId, msg.Location, msg.AssetId), false
	}

	ComponentTableId, success := GetComponentID(AssetTableID, *sC.ComponentName)

	// Changes should only be necessary between this marker

	txnStmtCtx, txnStmtCtxCl := context.WithDeadline(context.Background(), time.Now().Add(internal.FiveSeconds))
	// txnStmtCtxCl is the cancel function of the context, used in the statement creation.
	// It is deferred to automatically release the allocated resources, once the function returns
	defer txnStmtCtxCl()

	stmt := txn.StmtContext(txnStmtCtx, statement.InsertIntoMaintenanceActivities)

	stmtCtx, stmtCtxCl := context.WithDeadline(context.Background(), time.Now().Add(internal.FiveSeconds))
	// stmtCtxCl is the cancel function of the context, used in the transactions execution creation.
	// It is deferred to automatically release the allocated resources, once the function returns
	defer stmtCtxCl()

	_, err = stmt.ExecContext(stmtCtx, ComponentTableId, sC.Activity, sC.TimestampMs)
	if err != nil {

		if err != nil {
			pqErr := err.(*pq.Error)
			zap.S().Errorf("Error executing statement: %s -> %s", pqErr.Code, pqErr.Message)
			if pqErr.Code == "23P01" {
				return true, err, true
			}
			return true, err, false
		}
		zap.S().Debugf("Error inserting into addMaintenanceActivity table: %s", err.Error())
		return true, err, false
	}

	// And this marker

	if isDryRun {
		zap.S().Debugf("Dry run: not committing transaction")
		err = txn.Rollback()
		if err != nil {
			zap.S().Errorf("Error rolling back transaction: %s", err.Error())
			return true, err, false
		}
	} else {
		zap.S().Debugf("Committing transaction")
		err = txn.Commit()
		if err != nil {
			zap.S().Errorf("Error committing transaction: %s", err.Error())
			return true, err, false
		}
		isCommited = true
	}

	zap.S().Debugf("Successfully processed addMaintenanceActivity message: %v", msg)
	return false, err, false
}