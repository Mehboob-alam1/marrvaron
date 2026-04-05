package utils

import (
	"errors"
	"time"

	"marvaron/internal/config"
	"marvaron/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const registrationIssuer = "marvaron-registration"

type Claims struct {
	UserID uuid.UUID       `json:"user_id"`
	Email  string          `json:"email"`
	Role   models.UserRole `json:"role"`
	Roles  []string        `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken genera un JWT token per l'utente (active role + all roles)
func GenerateToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(config.AppConfig.GetJWTExpiry())
	roles := user.RolesAsStrings()

	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "marvaron",
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.AppConfig.JWT.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GenerateRegistrationToken short-lived JWT to continue signup (password + OTP steps)
func GenerateRegistrationToken(userID uuid.UUID) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Issuer:    registrationIssuer,
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	return token.SignedString([]byte(config.AppConfig.JWT.Secret))
}

// ParseRegistrationToken validates a registration-only JWT and returns user ID
func ParseRegistrationToken(tokenString string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(config.AppConfig.JWT.Secret), nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, errors.New("invalid registration token")
	}
	if claims.Issuer != registrationIssuer {
		return uuid.Nil, errors.New("invalid token type")
	}
	uid, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}
	return uid, nil
}

// ValidateToken valida e restituisce i claims del token
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(config.AppConfig.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.Issuer == registrationIssuer {
		return nil, errors.New("use full auth token, not registration token")
	}

	if len(claims.Roles) == 0 && claims.Role != "" {
		claims.Roles = []string{string(claims.Role)}
	}

	return claims, nil
}
