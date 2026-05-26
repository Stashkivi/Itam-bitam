package graph

import (
	"context"
	"fmt"

	"github.com/itam/server/internal/model"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// UpsertScan replaces the live dependency subgraph for one host with the
// data from the latest scan. The strategy is delete-and-recreate per host
// so stale nodes (terminated processes, closed ports) are never left behind.
func (c *Client) UpsertScan(ctx context.Context, scan *model.ScanPayload) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// 1. Merge the Host node (never deleted, survives scan cycles).
		if _, err := tx.Run(ctx, `
			MERGE (h:Host {uuid: $uuid})
			SET h.hostname  = $hostname,
			    h.os        = $os,
			    h.arch      = $arch,
			    h.updatedAt = datetime()`,
			map[string]interface{}{
				"uuid":     scan.HostUUID,
				"hostname": scan.Host.Hostname,
				"os":       scan.Host.OS,
				"arch":     scan.Host.Arch,
			}); err != nil {
			return nil, err
		}

		// 2. Detach-delete all previously computed relationships + child nodes
		//    for this host so we start from a clean slate.
		if _, err := tx.Run(ctx, `
			MATCH (h:Host {uuid: $uuid})-[*1..3]-(n)
			WHERE NOT n:Host
			DETACH DELETE n`,
			map[string]interface{}{"uuid": scan.HostUUID}); err != nil {
			return nil, err
		}

		// 3. Recreate Service → Process → Port chain.
		// Build a service-name → service map for port/process correlation.
		for _, svc := range scan.Services {
			if _, err := tx.Run(ctx, `
				MATCH (h:Host {uuid: $uuid})
				CREATE (s:Service {
				    name:   $name,
				    status: $status,
				    uuid:   $uuid + ':' + $name
				})
				CREATE (h)-[:RUNS]->(s)`,
				map[string]interface{}{
					"uuid":   scan.HostUUID,
					"name":   svc.Name,
					"status": svc.Status,
				}); err != nil {
				return nil, err
			}
		}

		// 4. Process nodes.
		for _, proc := range scan.Processes {
			if _, err := tx.Run(ctx, `
				MATCH (h:Host {uuid: $hostUUID})
				CREATE (p:Process {
				    pid:      $pid,
				    name:     $name,
				    exe:      $exe,
				    username: $username,
				    uuid:     $hostUUID + ':proc:' + toString($pid)
				})
				CREATE (h)-[:HAS_PROCESS]->(p)`,
				map[string]interface{}{
					"hostUUID": scan.HostUUID,
					"pid":      proc.PID,
					"name":     proc.Name,
					"exe":      proc.Exe,
					"username": proc.Username,
				}); err != nil {
				return nil, err
			}
		}

		// 5. Port nodes linked to their owning process.
		for _, port := range scan.OpenPorts {
			if _, err := tx.Run(ctx, `
				MATCH (h:Host {uuid: $hostUUID})-[:HAS_PROCESS]->(p:Process {pid: $pid})
				CREATE (pt:Port {
				    number:   $port,
				    protocol: $proto,
				    bindAddr: $bind,
				    uuid:     $hostUUID + ':' + $proto + ':' + toString($port)
				})
				CREATE (p)-[:LISTENS_ON]->(pt)`,
				map[string]interface{}{
					"hostUUID": scan.HostUUID,
					"pid":      port.PID,
					"port":     port.Port,
					"proto":    port.Protocol,
					"bind":     port.BindAddr,
				}); err != nil {
				// Process may have exited between scan phases — tolerate missing PID.
				continue
			}
		}

		return nil, nil
	})
	return err
}

// QueryDependencyMap returns the full dependency graph for one host as a
// serialisable map suitable for JSON encoding and frontend consumption.
func (c *Client) QueryDependencyMap(ctx context.Context, hostUUID string) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"host":      nil,
		"services":  []interface{}{},
		"processes": []interface{}{},
		"ports":     []interface{}{},
	}

	err := c.query(ctx, `
		MATCH (h:Host {uuid: $uuid})
		OPTIONAL MATCH (h)-[:RUNS]->(s:Service)
		OPTIONAL MATCH (h)-[:HAS_PROCESS]->(p:Process)
		OPTIONAL MATCH (p)-[:LISTENS_ON]->(pt:Port)
		RETURN h, collect(distinct s) AS services,
		       collect(distinct p)   AS processes,
		       collect(distinct pt)  AS ports`,
		map[string]interface{}{"uuid": hostUUID},
		func(record neo4j.Record) error {
			h, _ := record.Get("h")
			result["host"] = nodeProps(h)
			if svcs, ok := record.Get("services"); ok {
				result["services"] = collectionProps(svcs)
			}
			if procs, ok := record.Get("processes"); ok {
				result["processes"] = collectionProps(procs)
			}
			if ports, ok := record.Get("ports"); ok {
				result["ports"] = collectionProps(ports)
			}
			return nil
		},
	)
	return result, err
}

// QueryHostsWithService finds all host UUIDs running a named service.
// Useful for blast-radius analysis ("what breaks if nginx goes down?").
func (c *Client) QueryHostsWithService(ctx context.Context, serviceName string) ([]string, error) {
	var uuids []string
	err := c.query(ctx, `
		MATCH (h:Host)-[:RUNS]->(s:Service {name: $name})
		RETURN h.uuid AS uuid`,
		map[string]interface{}{"name": serviceName},
		func(record neo4j.Record) error {
			if v, ok := record.Get("uuid"); ok {
				uuids = append(uuids, fmt.Sprintf("%v", v))
			}
			return nil
		},
	)
	return uuids, err
}

// QueryUnknownPorts returns Port nodes whose numbers are not in the
// approvedPorts set. Used by the anomaly engine for UNKNOWN_PORT detection.
func (c *Client) QueryUnknownPorts(ctx context.Context, hostUUID string, approvedPorts []int64) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := c.query(ctx, `
		MATCH (h:Host {uuid: $uuid})-[:HAS_PROCESS]->(p:Process)-[:LISTENS_ON]->(pt:Port)
		WHERE NOT pt.number IN $approved
		RETURN pt.number AS port, pt.protocol AS proto,
		       pt.bindAddr AS bind, p.name AS process`,
		map[string]interface{}{"uuid": hostUUID, "approved": approvedPorts},
		func(record neo4j.Record) error {
			row := make(map[string]interface{})
			for _, k := range []string{"port", "proto", "bind", "process"} {
				if v, ok := record.Get(k); ok {
					row[k] = v
				}
			}
			results = append(results, row)
			return nil
		},
	)
	return results, err
}

// ---- helpers ----

func nodeProps(v interface{}) map[string]interface{} {
	if node, ok := v.(neo4j.Node); ok {
		return node.Props
	}
	return nil
}

func collectionProps(v interface{}) []interface{} {
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]interface{}, 0, len(list))
	for _, item := range list {
		if node, ok := item.(neo4j.Node); ok {
			out = append(out, node.Props)
		}
	}
	return out
}
