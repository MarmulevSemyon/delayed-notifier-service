package repository

import (
	"database/sql"
	"delayedNotifier/internal/model"
	"errors"
	"fmt"

	"github.com/wb-go/wbf/dbpg"
)

var (
	ErrNoSuchNotification      = errors.New("there is no notification with such id")
	ErrNotificationNotCanceled = errors.New("notification cannot be canceled")
)

type Repository struct {
	db *dbpg.DB
}

func New(db *dbpg.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) CreateNotification(notification *model.Notification) error {
	query := `
		INSERT INTO notifications (
			channel,
			recipient,
			message,
			send_at,
			status,
			attempts,
			max_attempts
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	err := r.db.Master.QueryRow(
		query,
		notification.Channel,
		notification.Recipient,
		notification.Message,
		notification.SendAt,
		notification.Status,
		notification.Attempts,
		notification.MaxAttempts,
	).Scan(
		&notification.ID,
		&notification.CreatedAt,
		&notification.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("could not create notification: %w", err)
	}

	return nil
}

func (r *Repository) GetNotificationByID(id int64) (*model.Notification, error) {
	query := `
		SELECT
			id,
			channel,
			recipient,
			message,
			send_at,
			status,
			attempts,
			max_attempts,
			last_error,
			canceled_at,
			created_at,
			updated_at
		FROM notifications
		WHERE id = $1
	`

	var notification model.Notification
	var lastError sql.NullString
	var canceledAt sql.NullTime

	err := r.db.Master.QueryRow(query, id).Scan(
		&notification.ID,
		&notification.Channel,
		&notification.Recipient,
		&notification.Message,
		&notification.SendAt,
		&notification.Status,
		&notification.Attempts,
		&notification.MaxAttempts,
		&lastError,
		&canceledAt,
		&notification.CreatedAt,
		&notification.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNoSuchNotification
		}
		return nil, fmt.Errorf("could not get notification from db: %w", err)
	}

	if lastError.Valid {
		notification.LastError = &lastError.String
	}
	if canceledAt.Valid {
		notification.CanceledAt = &canceledAt.Time
	}

	return &notification, nil
}

func (r *Repository) CancelNotificationByID(id int64) error {
	query := `
		UPDATE notifications
		SET
			status = $1,
			canceled_at = NOW(),
			updated_at = NOW()
		WHERE id = $2
		  AND status IN ($3, $4)
	`

	result, err := r.db.Master.Exec(
		query,
		model.StatusCanceled,
		id,
		model.StatusPending,
		model.StatusProcessing,
	)
	if err != nil {
		return fmt.Errorf("could not cancel notification: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get affected rows: %w", err)
	}

	if affected == 0 {
		return ErrNotificationNotCanceled
	}

	return nil
}

func (r *Repository) UpdateStatusByID(id int64, status model.NotificationStatus, lastError *string) error {
	query := `
		UPDATE notifications
		SET
			status = $1,
			last_error = $2,
			updated_at = NOW()
		WHERE id = $3
	`

	_, err := r.db.Master.Exec(query, status, lastError, id)
	if err != nil {
		return fmt.Errorf("could not update notification status: %w", err)
	}

	return nil
}

func (r *Repository) IncreaseAttemptsByID(id int64, lastError string) (int, int, error) {
	query := `
		UPDATE notifications
		SET
			attempts = attempts + 1,
			last_error = $1,
			updated_at = NOW()
		WHERE id = $2
		RETURNING attempts, max_attempts
	`

	var attempts int
	var maxAttempts int

	err := r.db.Master.QueryRow(query, lastError, id).Scan(&attempts, &maxAttempts)
	if err != nil {
		return 0, 0, fmt.Errorf("could not increase attempts: %w", err)
	}

	return attempts, maxAttempts, nil
}

func (r *Repository) ListNotifications() ([]model.Notification, error) {
	query := `
		SELECT
			id,
			channel,
			recipient,
			message,
			send_at,
			status,
			attempts,
			max_attempts,
			last_error,
			canceled_at,
			created_at,
			updated_at
		FROM notifications
		ORDER BY id DESC
	`

	rows, err := r.db.Master.Query(query)
	if err != nil {
		return nil, fmt.Errorf("could not list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []model.Notification

	for rows.Next() {
		var n model.Notification
		var lastError sql.NullString
		var canceledAt sql.NullTime

		err := rows.Scan(
			&n.ID,
			&n.Channel,
			&n.Recipient,
			&n.Message,
			&n.SendAt,
			&n.Status,
			&n.Attempts,
			&n.MaxAttempts,
			&lastError,
			&canceledAt,
			&n.CreatedAt,
			&n.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("could not scan notification: %w", err)
		}

		if lastError.Valid {
			n.LastError = &lastError.String
		}
		if canceledAt.Valid {
			n.CanceledAt = &canceledAt.Time
		}

		notifications = append(notifications, n)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return notifications, nil
}
