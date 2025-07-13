package database

import (
	"database/sql"
	"fmt"
	"notsofluffy-backend/internal/models"
)

type SettingsQueries struct {
	db *sql.DB
}

func NewSettingsQueries(db *sql.DB) *SettingsQueries {
	return &SettingsQueries{db: db}
}

func (q *SettingsQueries) GetAllSettings() ([]models.SiteSetting, error) {
	query := `
		SELECT id, key, value, description, created_at, updated_at
		FROM site_settings
		ORDER BY key
	`
	rows, err := q.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}
	defer rows.Close()

	var settings []models.SiteSetting
	for rows.Next() {
		var setting models.SiteSetting
		err := rows.Scan(
			&setting.ID,
			&setting.Key,
			&setting.Value,
			&setting.Description,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, setting)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate settings: %w", err)
	}

	return settings, nil
}

func (q *SettingsQueries) GetSettingByKey(key string) (*models.SiteSetting, error) {
	query := `
		SELECT id, key, value, description, created_at, updated_at
		FROM site_settings
		WHERE key = $1
	`
	setting := &models.SiteSetting{}
	err := q.db.QueryRow(query, key).Scan(
		&setting.ID,
		&setting.Key,
		&setting.Value,
		&setting.Description,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get setting %s: %w", key, err)
	}
	return setting, nil
}

func (q *SettingsQueries) UpdateSetting(key, value string) error {
	query := `
		UPDATE site_settings 
		SET value = $1, updated_at = CURRENT_TIMESTAMP
		WHERE key = $2
	`
	result, err := q.db.Exec(query, value, key)
	if err != nil {
		return fmt.Errorf("failed to update setting %s: %w", key, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("setting %s not found", key)
	}

	return nil
}

func (q *SettingsQueries) GetMaintenanceMode() (bool, error) {
	setting, err := q.GetSettingByKey("maintenance_mode")
	if err != nil {
		return false, err
	}
	if setting == nil {
		return false, nil
	}
	return setting.Value == "true", nil
}