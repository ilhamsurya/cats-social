package repository

import (
	"context"
	"database/sql"
	"projectsphere/cats-social/internal/user/entity"
	"projectsphere/cats-social/pkg/database"
	"projectsphere/cats-social/pkg/protocol/msg"
	"strings"
)

type UserRepo struct {
	dbConnector database.PostgresConnector
}

func NewUserRepo(dbConnector database.PostgresConnector) UserRepo {
	return UserRepo{
		dbConnector: dbConnector,
	}
}

func (r UserRepo) CreateUser(ctx context.Context, param entity.UserParam) (entity.User, error) {
	query := `
		INSERT INTO users (email, name, password, salt) VALUES 
		($1, $2, $3, $4) RETURNING id_user, email, name, password, salt, created_at, updated_at
	`

	var row entity.User
	err := r.dbConnector.DB.GetContext(
		ctx,
		&row,
		query,
		param.Email,
		param.Name,
		param.Password,
		param.Salt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return entity.User{}, &msg.RespError{
				Code:    409,
				Message: msg.ErrEmailAlreadyExist,
			}
		} else {
			return entity.User{}, msg.InternalServerError(err.Error())
		}
	}

	return row, nil
}

func (r UserRepo) GetUserByEmail(ctx context.Context, email string) (entity.User, error) {
	query := `
		SELECT id_user, email, name, password, salt, created_at, updated_at FROM users WHERE email = $1
	`

	var row entity.User
	err := r.dbConnector.DB.GetContext(
		ctx,
		&row,
		query,
		email,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.User{}, msg.NotFound(msg.ErrUserNotFound)
		} else {
			return entity.User{}, msg.InternalServerError(err.Error())
		}
	}

	return row, nil
}

func (r UserRepo) IsUserExist(ctx context.Context, userId uint32) bool {
	query := `
		SELECT id_user, email, name, password, salt, created_at, updated_at FROM users WHERE id_user = $1
	`

	var row entity.User
	err := r.dbConnector.DB.GetContext(
		ctx,
		&row,
		query,
		userId,
	)
	if err != nil {
		return false
	}

	return true
}
