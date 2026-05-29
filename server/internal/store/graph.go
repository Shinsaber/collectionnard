package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
)

func (s *PgStore) GetItemGraph(ctx context.Context, itemID uuid.UUID, maxDepth int, relationTypeIDs []uuid.UUID) (*model.GraphResponse, error) {
	rtFilter := ""
	args := []interface{}{itemID, maxDepth}
	if len(relationTypeIDs) > 0 {
		placeholders := make([]string, len(relationTypeIDs))
		for i, id := range relationTypeIDs {
			args = append(args, id)
			placeholders[i] = fmt.Sprintf("$%d", i+3)
		}
		rtFilter = fmt.Sprintf("AND ir.relation_type_id IN (%s)", strings.Join(placeholders, ","))
	}

	query := fmt.Sprintf(`
		WITH RECURSIVE graph_down AS (
			SELECT i.id, i.item_type_id, i.attributes, it.name AS type_name, 0 AS depth,
			       ARRAY[i.id] AS path
			FROM items i
			JOIN item_types it ON it.id = i.item_type_id
			WHERE i.id = $1

			UNION ALL

			SELECT i2.id, i2.item_type_id, i2.attributes, it2.name AS type_name, g.depth + 1,
			       g.path || i2.id
			FROM graph_down g
			JOIN item_relations ir ON ir.source_item_id = g.id %[1]s
			JOIN items i2 ON i2.id = ir.target_item_id
			JOIN item_types it2 ON it2.id = i2.item_type_id
			WHERE g.depth < $2
			  AND NOT i2.id = ANY(g.path)
		),
		graph_up AS (
			SELECT i.id, i.item_type_id, i.attributes, it.name AS type_name, 0 AS depth,
			       ARRAY[i.id] AS path
			FROM items i
			JOIN item_types it ON it.id = i.item_type_id
			WHERE i.id = $1

			UNION ALL

			SELECT i2.id, i2.item_type_id, i2.attributes, it2.name AS type_name, g.depth - 1,
			       g.path || i2.id
			FROM graph_up g
			JOIN item_relations ir ON ir.target_item_id = g.id %[1]s
			JOIN items i2 ON i2.id = ir.source_item_id
			JOIN item_types it2 ON it2.id = i2.item_type_id
			WHERE g.depth > -$2
			  AND NOT i2.id = ANY(g.path)
		),
		graph AS (
			SELECT * FROM graph_down
			UNION ALL
			SELECT * FROM graph_up WHERE depth != 0
		)
		SELECT DISTINCT ON (id) id, type_name, attributes, depth
		FROM graph
		ORDER BY id, ABS(depth)`, rtFilter)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resp := &model.GraphResponse{
		Nodes: []model.GraphNode{},
		Edges: []model.GraphEdge{},
	}

	var foundRoot bool
	for rows.Next() {
		var node model.GraphNode
		var attrs json.RawMessage
		if err := rows.Scan(&node.ID, &node.Type, &attrs, &node.Depth); err != nil {
			return nil, err
		}
		node.Attributes = attrs
		node.Label = extractLabel(attrs)

		if node.ID == itemID && !foundRoot {
			resp.Root = node
			foundRoot = true
		} else {
			resp.Nodes = append(resp.Nodes, node)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !foundRoot {
		return nil, ErrNotFound
	}

	// Fetch edges between discovered nodes
	allIDs := []uuid.UUID{resp.Root.ID}
	for _, n := range resp.Nodes {
		allIDs = append(allIDs, n.ID)
	}

	edgeQuery := `
		SELECT ir.source_item_id, ir.target_item_id, rt.name, ir.attributes
		FROM item_relations ir
		JOIN relation_types rt ON rt.id = ir.relation_type_id
		WHERE ir.source_item_id = ANY($1) AND ir.target_item_id = ANY($1)`
	edgeArgs := []interface{}{allIDs}

	if len(relationTypeIDs) > 0 {
		edgeQuery += ` AND ir.relation_type_id = ANY($2)`
		edgeArgs = append(edgeArgs, relationTypeIDs)
	}

	edgeRows, err := s.pool.Query(ctx, edgeQuery, edgeArgs...)
	if err != nil {
		return nil, err
	}
	defer edgeRows.Close()

	for edgeRows.Next() {
		var edge model.GraphEdge
		if err := edgeRows.Scan(&edge.SourceID, &edge.TargetID, &edge.RelationType, &edge.Attributes); err != nil {
			return nil, err
		}
		resp.Edges = append(resp.Edges, edge)
	}

	return resp, edgeRows.Err()
}

func extractLabel(attrs json.RawMessage) string {
	var m map[string]interface{}
	if err := json.Unmarshal(attrs, &m); err != nil {
		return ""
	}
	for _, key := range []string{"titre", "nom", "name", "title", "numero"} {
		if v, ok := m[key]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}
