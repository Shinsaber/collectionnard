package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/label"
	"github.com/shinsaber/colec/internal/model"
)

// GetItemGraph explores the relation graph around itemID with a breadth-first
// search, level by level and in both directions (outgoing relations give a
// positive depth, incoming ones a negative depth). Each node is visited once
// (the first time it is reached, hence with its smallest |depth|), so the cost
// is bounded by the number of distinct nodes, not by the number of paths.
// Exploration stops at maxDepth levels or maxNodes nodes; in the latter case
// Truncated is set on the response.
func (s *PgStore) GetItemGraph(ctx context.Context, itemID uuid.UUID, maxDepth, maxNodes int, relationTypeIDs []uuid.UUID) (*model.GraphResponse, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM items WHERE id = $1)`, itemID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	if maxNodes < 1 {
		maxNodes = 1
	}

	depths := map[uuid.UUID]int{itemID: 0}
	order := []uuid.UUID{itemID}
	truncated := false

	down := []uuid.UUID{itemID}
	up := []uuid.UUID{itemID}
	for level := 1; level <= maxDepth && (len(down) > 0 || len(up) > 0) && !truncated; level++ {
		var err error
		down, err = s.expandLevel(ctx, down, true, level, relationTypeIDs, depths, &order, maxNodes, &truncated)
		if err != nil {
			return nil, err
		}
		if truncated {
			break
		}
		up, err = s.expandLevel(ctx, up, false, level, relationTypeIDs, depths, &order, maxNodes, &truncated)
		if err != nil {
			return nil, err
		}
	}

	resp := &model.GraphResponse{
		Nodes:     []model.GraphNode{},
		Edges:     []model.GraphEdge{},
		Truncated: truncated,
	}

	// Node details in one query.
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, it.name, i.attributes, it.attributes_schema
		FROM items i
		JOIN item_types it ON it.id = i.item_type_id
		WHERE i.id = ANY($1)`, order)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	nodesByID := make(map[uuid.UUID]model.GraphNode, len(order))
	for rows.Next() {
		var node model.GraphNode
		var attrs, schema json.RawMessage
		if err := rows.Scan(&node.ID, &node.Type, &attrs, &schema); err != nil {
			return nil, err
		}
		node.Attributes = attrs
		node.Label = label.Render(schema, attrs, label.Langs[0])
		node.Depth = depths[node.ID]
		nodesByID[node.ID] = node
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range order { // discovery order, root first
		node, ok := nodesByID[id]
		if !ok {
			continue // deleted between the traversal and the fetch
		}
		if id == itemID {
			resp.Root = node
		} else {
			resp.Nodes = append(resp.Nodes, node)
		}
	}

	// Edges between discovered nodes.
	edgeQuery := `
		SELECT ir.source_item_id, ir.target_item_id, rt.name, ir.attributes
		FROM item_relations ir
		JOIN relation_types rt ON rt.id = ir.relation_type_id
		WHERE ir.source_item_id = ANY($1) AND ir.target_item_id = ANY($1)`
	edgeArgs := []any{order}
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

// expandLevel fetches the neighbours of frontier (targets of outgoing relations
// when down, sources of incoming ones otherwise), records the new ones at the
// given depth and returns them as the next frontier.
func (s *PgStore) expandLevel(ctx context.Context, frontier []uuid.UUID, down bool, level int, relationTypeIDs []uuid.UUID,
	depths map[uuid.UUID]int, order *[]uuid.UUID, maxNodes int, truncated *bool) ([]uuid.UUID, error) {
	if len(frontier) == 0 {
		return nil, nil
	}
	from, to := "source_item_id", "target_item_id"
	if !down {
		from, to = to, from
	}
	query := fmt.Sprintf(`SELECT DISTINCT %s FROM item_relations WHERE %s = ANY($1)`, to, from)
	args := []any{frontier}
	if len(relationTypeIDs) > 0 {
		query += ` AND relation_type_id = ANY($2)`
		args = append(args, relationTypeIDs)
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var next []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if _, seen := depths[id]; seen {
			continue
		}
		if len(*order) >= maxNodes {
			*truncated = true
			break
		}
		depth := level
		if !down {
			depth = -level
		}
		depths[id] = depth
		*order = append(*order, id)
		next = append(next, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return next, nil
}
