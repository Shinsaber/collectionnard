package store

import (
	"context"

	"github.com/shinsaber/colec/internal/model"
)

func (s *PgStore) GetStats(ctx context.Context) (*model.Stats, error) {
	var stats model.Stats

	// Items by type
	rows, err := s.pool.Query(ctx,
		`SELECT it.name, count(i.id)
		 FROM item_types it
		 LEFT JOIN items i ON i.item_type_id = it.id
		 GROUP BY it.name
		 ORDER BY it.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ibt model.ItemsByType
		if err := rows.Scan(&ibt.TypeName, &ibt.Count); err != nil {
			return nil, err
		}
		stats.ItemsByType = append(stats.ItemsByType, ibt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	err = s.pool.QueryRow(ctx, `SELECT count(*) FROM items`).Scan(&stats.TotalItems)
	if err != nil {
		return nil, err
	}

	err = s.pool.QueryRow(ctx, `SELECT count(*) FROM collections`).Scan(&stats.TotalCollections)
	if err != nil {
		return nil, err
	}

	err = s.pool.QueryRow(ctx, `SELECT count(*) FROM item_relations`).Scan(&stats.TotalRelations)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}
