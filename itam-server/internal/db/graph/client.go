// Package graph manages the live service dependency graph in Neo4j/Memgraph.
// The Bolt protocol is identical between the two; switching is a DSN change.
package graph

import (
	"context"
	"fmt"

	"github.com/itam/server/internal/config"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps the Neo4j driver session factory.
type Client struct {
	driver neo4j.DriverWithContext
}

// Connect opens a verified connection to Neo4j or Memgraph.
func Connect(ctx context.Context, cfg *config.GraphConfig) (*Client, error) {
	auth := neo4j.BasicAuth(cfg.Username, cfg.Password, "")
	driver, err := neo4j.NewDriverWithContext(cfg.URI, auth)
	if err != nil {
		return nil, fmt.Errorf("graph driver: %w", err)
	}
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("graph connectivity: %w", err)
	}
	return &Client{driver: driver}, nil
}

// Close releases the driver and all open sessions.
func (c *Client) Close(ctx context.Context) { c.driver.Close(ctx) } //nolint:errcheck

// run executes a write transaction with the supplied Cypher and parameters.
func (c *Client) run(ctx context.Context, cypher string, params map[string]interface{}) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, cypher, params)
		return nil, err
	})
	return err
}

// query executes a read transaction and calls collector for each result record.
func (c *Client) query(ctx context.Context, cypher string, params map[string]interface{},
	collector func(neo4j.Record) error) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		for result.Next(ctx) {
			if err := collector(*result.Record()); err != nil {
				return nil, err
			}
		}
		return nil, result.Err()
	})
	return err
}
