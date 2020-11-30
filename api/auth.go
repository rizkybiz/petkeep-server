package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

//GenerateToken creates a JWT token from user ID and expires in one hour
func generateToken(userID uint, exp time.Time) (string, error) {

	//Generate Token Claims
	accessClaims := jwt.MapClaims{}
	accessClaims["jti"] = strconv.FormatUint(uint64(userID), 10)
	accessClaims["exp"] = exp
	accessClaims["iss"] = "petkeep-server"
	//Generate token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	//Return token string signed by secret signing key
	return accessToken.SignedString([]byte(jwtSigningKey))
}

//ValidateToken validates that a received token is in fact valid
func validateToken(r *http.Request) (int64, error) {
	tokenStr := extractToken(r)
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return 0, fmt.Errorf("internal server error")
		}
		return []byte(jwtSigningKey), nil
	})
	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		uid, err := strconv.Atoi(fmt.Sprintf("%v", claims["jti"]))
		if err != nil {
			return 0, err
		}
		return int64(uid), nil
	}
	return 0, nil
}

//extractToken is a helper function for getting the token string from HTTP headers
func extractToken(r *http.Request) string {
	keys := r.URL.Query()
	token := keys.Get("token")
	if token != "" {
		return token
	}
	bearerToken := r.Header.Get("Authorization")
	if len(strings.Split(bearerToken, " ")) == 2 {
		return strings.Split(bearerToken, " ")[1]
	}
	return bearerToken
}
