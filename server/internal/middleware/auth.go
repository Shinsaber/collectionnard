package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/shinsaber/colec/internal/model"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	RoleKey   contextKey = "role"
)

// Issuer is the `iss` claim stamped on every access token.
const Issuer = "collectionnard"

// leeway absorbs small clock drifts between the issuer and the verifier.
const leeway = 30 * time.Second

func GetUserID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(UserIDKey).(uuid.UUID)
	return id
}

func GetRole(ctx context.Context) string {
	role, _ := ctx.Value(RoleKey).(string)
	return role
}

// Claims are the JWT claims carried by an access token.
type Claims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

// Auth issues and verifies short-lived HS256 access tokens.
type Auth struct {
	secret    []byte
	accessTTL time.Duration
	parser    *jwt.Parser
}

// NewAuth builds an Auth that signs tokens with secret and makes them expire after accessTTL.
func NewAuth(secret string, accessTTL time.Duration) *Auth {
	return &Auth{
		secret:    []byte(secret),
		accessTTL: accessTTL,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithExpirationRequired(),
			jwt.WithIssuer(Issuer),
			jwt.WithLeeway(leeway),
		),
	}
}

// AccessTTL returns the lifetime of the access tokens issued by GenerateToken.
func (a *Auth) AccessTTL() time.Duration {
	return a.accessTTL
}

// Authenticate verifies the JWT token and injects user info into context.
func (a *Auth) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeUnauthorized(w, "missing token")
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		if tokenStr == header {
			writeUnauthorized(w, "Bearer token expected")
			return
		}

		claims, err := a.ParseToken(tokenStr)
		if err != nil {
			writeUnauthorized(w, "invalid or expired token")
			return
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			writeUnauthorized(w, "invalid token subject")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin rejects non-admin users with 403.
func (a *Auth) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetRole(r.Context()) != "admin" {
			writeError(w, http.StatusForbidden, "forbidden", "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ParseToken verifies signature, algorithm, issuer and expiration of an access token.
func (a *Auth) ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := a.parser.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return a.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

// GenerateToken creates a signed access token for the given user.
func (a *Auth) GenerateToken(userID uuid.UUID, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(a.accessTTL)),
		},
		Role: role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secret)
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="collectionnard"`)
	writeError(w, http.StatusUnauthorized, "unauthorized", message)
}

// writeError emits a model.APIError as JSON (same shape as the handler package).
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.APIError{Error: code, Message: message})
}
