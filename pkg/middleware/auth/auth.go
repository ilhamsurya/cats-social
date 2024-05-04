package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"projectsphere/cats-social/pkg/protocol/msg"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var JWT_SIGNING_METHOD = jwt.SigningMethodHS256

type JWTAuth struct {
	ExpireTimeInMinute int
	SecretKey          string
	IsAuthorizedUser   func(context.Context, uint32) bool
}

func NewJwtAuth(expireTimeInMinute int, secretKey string, isAuthorizedUser func(context.Context, uint32) bool) JWTAuth {
	return JWTAuth{
		ExpireTimeInMinute: expireTimeInMinute,
		SecretKey:          secretKey,
		IsAuthorizedUser:   isAuthorizedUser,
	}
}

func (j JWTAuth) GenerateToken(userId uint32) (string, error) {
	now := time.Now()

	expiredTokenTime := jwt.NewNumericDate(
		now.Add(
			time.Duration(j.ExpireTimeInMinute) * time.Minute,
		),
	)

	claims := jwt.MapClaims{}
	claims["userId"] = userId
	// Issued At
	claims["iat"] = now
	// Expiration Time
	claims["exp"] = expiredTokenTime

	token := jwt.NewWithClaims(JWT_SIGNING_METHOD, claims)
	signedToken, err := token.SignedString([]byte(j.SecretKey))
	if err != nil {
		return "", msg.InternalServerError(err.Error())
	}

	return signedToken, nil
}

func (j JWTAuth) TokenValid(c *gin.Context) error {
	tokenString := ExtractToken(c)
	if tokenString == "" {
		return errors.New(msg.ErrTokenNotExist)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if method, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, &msg.RespError{
				Code:    http.StatusUnauthorized,
				Message: msg.ErrInvalidSigningMethod,
			}
		} else if method != JWT_SIGNING_METHOD {
			return nil, &msg.RespError{
				Code:    http.StatusUnauthorized,
				Message: msg.ErrInvalidSigningMethod,
			}
		}
		return []byte(j.SecretKey), nil
	})

	if err != nil {
		return &msg.RespError{
			Code:    http.StatusUnauthorized,
			Message: err.Error(),
		}
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		switch {
		case errors.Is(err, jwt.ErrTokenMalformed):
			return &msg.RespError{
				Code:    http.StatusUnauthorized,
				Message: msg.ErrInvalidToken,
			}
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return &msg.RespError{
				Code:    http.StatusUnauthorized,
				Message: msg.ErrInvalidSigningMethod,
			}
		case errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet):
			return &msg.RespError{
				Code:    http.StatusUnauthorized,
				Message: msg.ErrTokenAlreadyExpired,
			}
		default:
			return &msg.RespError{
				Code:    http.StatusUnauthorized,
				Message: err.Error(),
			}
		}
	}

	uid, err := strconv.ParseUint(fmt.Sprintf("%.0f", claims["userId"]), 10, 32)
	if err != nil {
		return &msg.RespError{
			Code:    http.StatusUnauthorized,
			Message: err.Error(),
		}
	}

	userId := uint32(uid)

	if !j.IsAuthorizedUser(c.Request.Context(), userId) {
		return &msg.RespError{
			Code:    http.StatusUnauthorized,
			Message: msg.ErrInvalidToken,
		}
	}

	return nil
}

func ExtractToken(c *gin.Context) string {
	token := c.Query("token")
	if token != "" {
		return token
	}

	bearerToken := c.Request.Header.Get("Authorization")
	if len(strings.Split(bearerToken, " ")) == 2 {
		return strings.Split(bearerToken, " ")[1]
	}

	return ""
}

func (j JWTAuth) JwtAuthUserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := j.TokenValid(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, msg.Unauthorization(err.Error()))
			c.Abort()
			return
		}
		c.Next()
	}
}
