package jwt

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gonstruct/core/facades/authentication"

	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	Issuer       string
	Secret       string
	Audience     []string
	ParseSession func(id string, arg json.RawMessage) (authentication.Session, error)
}

type jwtClaims struct {
	jwt.RegisteredClaims

	Session json.RawMessage
}

func (s Service) GenerateToken(session authentication.Session) (authentication.Token, error) {
	sessionData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	claims := &jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        session.ID(),
			Issuer:    s.Issuer,
			Audience:  jwt.ClaimStrings(s.Audience),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(session.Lifetime())),
		},
		Session: sessionData,
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.Secret))
	if err != nil {
		return nil, err
	}

	return jwtToken{
		id:        claims.ID,
		signed:    signed,
		expiresAt: claims.ExpiresAt.Time,
	}, nil
}

func (s Service) ValidateToken(token string, strict ...bool) (authentication.Session, error) {
	var claims jwtClaims
	parsedToken, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(token *jwt.Token) (interface{}, error) { return []byte(s.Secret), nil },
	)
	if err != nil {
		return nil, err
	}

	if !parsedToken.Valid || (len(strict) == 1 && strict[0] && claims.Issuer != s.Issuer) {
		return nil, fmt.Errorf("token is malformed or expired | valid: %v | issuer: %s, expected: %s", parsedToken.Valid, claims.Issuer, s.Issuer)
	}

	return s.ParseSession(claims.ID, claims.Session)
}

type jwtToken struct {
	id        string
	signed    string
	expiresAt time.Time
}

func (t jwtToken) ID() string {
	return t.id
}

func (t jwtToken) String() string {
	return t.signed
}

func (t jwtToken) ExpiresAt() time.Time {
	return t.expiresAt
}
