package data

import (
	"context"
	"database/sql"
	"time"
)

// Permissions represents a list of permission codes. A user may have multiple permissions.
type Permissions []string

// Include returns true if the given permission code is present in the permissions slice.
func (p Permissions) Include(code string) bool {
	// Just use slices.Contains?
	for i := range p {
		if code == p[i] {
			return true
		}
	}
	return false
}

type PermissionModel struct {
	DB *sql.DB
}

// GetAllForUser returns all permissions granted to a specific user. It uses the same pattern as MovieModel.GetAll.
func (m PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
SELECT permissions.code
FROM permissions
INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
INNER JOIN users ON users_permissions.user_id = users.id
WHERE users.id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions Permissions
	for rows.Next() {
		var permission string
		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}
