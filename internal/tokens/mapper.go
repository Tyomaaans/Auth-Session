package tokens

import (
	"auth-session/internal/domains"
)

// Entity to Response

func ToTokenPairResponse(e domains.TokenPairEntity) *TokenPairResponse {
	return &TokenPairResponse{
		UserID:       e.UserID,
		AccessToken:  e.AccessToken,
		RefreshToken: e.RefreshToken,
	}
}