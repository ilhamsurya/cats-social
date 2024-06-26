package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"projectsphere/cats-social/internal/cat/entity"
	"projectsphere/cats-social/pkg/database"
	"projectsphere/cats-social/pkg/protocol/msg"
	"strconv"
)

type CatRepo struct {
	dbConnector database.PostgresConnector
}

func NewCatRepo(dbConnector database.PostgresConnector) CatRepo {
	return CatRepo{
		dbConnector: dbConnector,
	}
}

func (r CatRepo) CreateCat(ctx context.Context, param entity.CatParam, userId uint32) (entity.Cat, error) {
	var cat entity.Cat
	query := `
        INSERT INTO "cats" (name, race, sex, age_in_month, description,id_user)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id_cat
    `

	err := r.dbConnector.DB.QueryRow(query, param.Name,
		param.Race,
		param.Sex,
		param.AgeInMonth,
		param.Description, int(userId)).Scan(&cat.IdCat)

	if err != nil {
		if err == sql.ErrNoRows {
			return entity.Cat{}, msg.BadRequest("no rows were returned")
		}
		return entity.Cat{}, msg.InternalServerError(err.Error())
	}

	// Insert images into cat_image table
	for _, imageURL := range param.ImageURLs {
		_, err := r.insertCatImage(ctx, cat.IdCat, imageURL)
		if err != nil {
			return entity.Cat{}, msg.InternalServerError(err.Error())
		}
	}

	cat.Name = param.Name
	cat.Race = param.Race
	cat.Sex = param.Sex
	cat.AgeInMonth = param.AgeInMonth
	cat.Description = param.Description

	return cat, nil
}

func (r CatRepo) insertCatImage(ctx context.Context, catID uint32, imageURL string) (int, error) {
	query := `
		INSERT INTO cat_images (id_cat, image)
		VALUES ($1, $2)
		RETURNING id_image
	`

	var idImage int
	err := r.dbConnector.DB.QueryRowContext(
		ctx,
		query,
		catID,
		imageURL,
	).Scan(
		&idImage,
	)

	if err != nil {
		return 0, err
	}

	return idImage, nil
}

// UpdateCat updates the cat information in the database.
func (r CatRepo) UpdateCat(ctx context.Context, catID int, catParam entity.CatParam) (entity.Cat, error) {
	query := `
		UPDATE cats
		SET name = $1, race = $2, sex = $3, age_in_month = $4, description = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id_cat = $6
		RETURNING id_cat, name, race, sex, age_in_month, description, created_at, updated_at
	`

	fmt.Print(catID, catParam)
	// Execute the update query
	row := r.dbConnector.DB.QueryRowContext(ctx, query, catParam.Name, catParam.Race, catParam.Sex, catParam.AgeInMonth, catParam.Description, catID)

	// Scan the updated cat from the database row
	var updatedCat entity.Cat
	err := row.Scan(&updatedCat.IdCat, &updatedCat.Name, &updatedCat.Race, &updatedCat.Sex, &updatedCat.AgeInMonth, &updatedCat.Description, &updatedCat.CreatedAt, &updatedCat.UpdatedAt)
	if err != nil {
		return entity.Cat{}, err
	}

	return updatedCat, nil
}

func (r CatRepo) GetUserCatGender(ctx context.Context, catID int) (string, error) {
	var gender string
	query := `
        SELECT sex FROM "cats" WHERE id_cat = $1
    `
	err := r.dbConnector.DB.QueryRowContext(ctx, query, catID).Scan(&gender)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("404: Cat not found")
		}
		return "", errors.New("500: Internal server error")
	}

	return gender, nil
}

func (r CatRepo) GetCatByID(ctx context.Context, catID int) (entity.Cat, error) {
	var cat entity.Cat
	query := `
        SELECT id_cat, id_user, name, race, sex, age_in_month, description
        FROM "cats" WHERE id_cat = $1
    `
	err := r.dbConnector.DB.QueryRowContext(ctx, query, catID).Scan(
		&cat.IdCat, &cat.IdUser, &cat.Name, &cat.Race, &cat.Sex, &cat.AgeInMonth, &cat.Description)
	fmt.Println(err)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.Cat{}, errors.New("404: Cat not found")
		}
		return entity.Cat{}, errors.New("500: Internal server error")
	}

	return cat, nil
}

func (r CatRepo) CatExists(ctx context.Context, catID int) bool {
	query := `
        SELECT EXISTS (SELECT 1 FROM "cats" WHERE id_cat = $1)
    `
	var exists bool
	err := r.dbConnector.DB.QueryRowContext(ctx, query, catID).Scan(&exists)
	if err != nil {
		// Handle error, log or return false
		return false
	}

	return exists
}

func (r CatRepo) GetCatOwner(ctx context.Context, catID int) (int, error) {
	var ownerID int
	query := `
        SELECT id_user FROM "cats" WHERE id_cat = $1
    `
	err := r.dbConnector.DB.QueryRowContext(ctx, query, catID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("404: Cat not found")
		}
		return 0, errors.New("500: Internal server error")
	}

	return ownerID, nil
}

func (r CatRepo) IsUserCatAssociationValid(ctx context.Context, userID, catID int) (bool, error) {
	var count int
	query := `
        SELECT COUNT(*) FROM "cats" WHERE id_user = $1 AND id_cat = $2
    `
	err := r.dbConnector.DB.QueryRowContext(ctx, query, userID, catID).Scan(&count)
	if err != nil {
		return false, err // Handle error
	}

	return count > 0, nil
}

func (r CatRepo) GetCat(ctx context.Context, param entity.GetCatParam, ageOperator string, age int) ([]entity.Cat, error) {
	// query := `
	// 	SELECT
	// 		c.id_cat, c.name, c.race, c.sex, c.age_in_month, c.description,
	// 		ci.id_image, ci.id_cat, ci.image,
	// 		mc.id_match, mc.is_matched
	// 	FROM "cats" c
	// 	JOIN "cat_images" ci ON ci.id_cat = c.id_cat
	// 	JOIN "users" u ON u.id_user = c.id_user
	// 	LEFT JOIN "match_cats" mc ON mc.id_user_cat = c.id_cat
	// 	WHERE c.deleted_at IS NULL
	// `

	query := `
		SELECT 
			c.id_cat, c.name, c.race, c.sex, c.age_in_month, c.description, c.id_user, c.deleted_at
		FROM "cats" as c
		WHERE c.deleted_at IS NULL 
	`

	args := []interface{}{}
	argsCount := 1

	if param.IdCat != nil {
		query += fmt.Sprintf(" AND c.id_cat = $%d", argsCount)
		args = append(args, &param.IdCat)
		argsCount++
	}
	if param.Race != "" {
		query += fmt.Sprintf(" AND c.race = $%d", argsCount)
		args = append(args, &param.Race)
		argsCount++
	}
	if param.AgeInMonth != "" {
		switch ageOperator {
		case ">":
			query += fmt.Sprintf(" AND c.age_in_month > $%d", argsCount)
		case "<":
			query += fmt.Sprintf(" AND c.age_in_month < $%d", argsCount)
		default:
			query += fmt.Sprintf(" AND c.age_in_month = $%d", argsCount)
		}
		args = append(args, age)
		argsCount++
	}
	if param.Owned != nil {
		query += fmt.Sprintf(" AND c.id_user = $%d", argsCount)
		args = append(args, &param.IdUser)
		fmt.Println(&param.IdUser)
		argsCount++
	}
	if param.Sex != "" {
		query += fmt.Sprintf(" AND c.sex = $%d", argsCount)
		args = append(args, &param.Sex)
		argsCount++
	}
	if param.Search != "" {
		query += " AND c.name ILIKE '%' || $" + strconv.Itoa(argsCount) + " || '%'"
		args = append(args, &param.Search)
		argsCount++
	}

	if param.Limit != nil {
		query += fmt.Sprintf(" LIMIT $%d", argsCount)
		args = append(args, *param.Limit)
		argsCount++
	}

	if param.Offset != nil {
		query += fmt.Sprintf(" OFFSET $%d", argsCount)
		args = append(args, &param.Offset)
		argsCount++
	}

	dataQuery := fmt.Sprintf(`
		SELECT 
			dc.id_cat, dc.name, dc.race, dc.sex, dc.age_in_month, dc.description, 
			ci.id_image, ci.id_cat, ci.image, 
			mc.id_match, mc.approved_at
		FROM (%s) dc
		JOIN "cat_images" ci ON ci.id_cat = dc.id_cat 
		JOIN "users" u ON u.id_user = dc.id_user
		LEFT JOIN "match_cats" mc ON mc.id_user_cat = dc.id_cat
		WHERE dc.deleted_at IS NULL 
	`, query)

	if param.HasMatched != nil {
		if *param.HasMatched {
			dataQuery += " AND mc.approved_at IS NOT NULL"
		} else {
			dataQuery += " AND (mc.rejected_at IS NULL AND mc.approved_at IS NULL)"
		}
	}

	rows, err := r.dbConnector.DB.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return []entity.Cat{}, msg.InternalServerError(err.Error())
	}
	defer rows.Close()

	var cats []entity.Cat
	var catsMap = make(map[int]entity.Cat)

	for rows.Next() {
		var cat = entity.Cat{}
		var image = entity.CatImage{}
		var idMatch *uint32
		var approvedAt sql.NullTime
		var err = rows.Scan(
			&cat.IdCat, &cat.Name, &cat.Race, &cat.Sex, &cat.AgeInMonth, &cat.Description,
			&image.IdImage, &image.IdCat, &image.Image,
			&idMatch, &approvedAt,
		)

		if err != nil {
			return []entity.Cat{}, msg.InternalServerError(err.Error())
		}

		catTemp, ok := catsMap[int(cat.IdCat)]
		if !ok {
			catTemp = entity.Cat{
				IdCat:       cat.IdCat,
				Name:        cat.Name,
				Race:        cat.Race,
				Sex:         cat.Sex,
				AgeInMonth:  cat.AgeInMonth,
				Description: cat.Description,
				CreatedAt:   cat.CreatedAt,
				UpdatedAt:   cat.UpdatedAt,
				CatImage:    make([]entity.CatImage, 0),
				MatchCat:    make([]entity.MatchCat, 0),
			}
		}
		catTemp.CatImage = append(catTemp.CatImage, image)
		if idMatch != nil {
			catTemp.MatchCat = append(catTemp.MatchCat, entity.MatchCat{IdMatch: *idMatch, ApprovedAt: approvedAt})
		}
		catsMap[int(cat.IdCat)] = catTemp
	}

	for _, cat := range catsMap {
		cats = append(cats, cat)
	}

	return cats, nil
}

func (r CatRepo) DeleteCat(ctx context.Context, catID int, userID int) error {
	query := `
		UPDATE cats SET deleted_at = NOW() 
		WHERE id_cat = $1 
		AND id_user = $2
		AND deleted_at IS NULL
	`

	res, err := r.dbConnector.DB.ExecContext(ctx, query, catID, userID)
	if err != nil {
		return msg.InternalServerError(err.Error())
	}

	rowEffect, err := res.RowsAffected()
	if err != nil {
		return msg.InternalServerError(err.Error())
	}

	if rowEffect == 0 {
		return msg.NotFound("id is not found")
	}

	return nil
}
