package users

import (
	"context"
	"encoding/base64"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"auth-session/internal/domains"
	"auth-session/internal/sessions"
	"auth-session/internal/tokens"
	"auth-session/pkg"

	jsonValidator "auth-session/pkg"
)

// Interface

type UserService interface {
	RegisterUser(ctx context.Context, req RegisterUserRequest) error
	UpdateUser(ctx context.Context, update *UpdateUserRequest) (*UserResponse, error)
	GetUserByID(ctx context.Context, userID string) (*UserResponse, error)
	GetUsers(ctx context.Context) ([]UserResponse, error)
	LoginUser(ctx context.Context, agent, ip string, req LoginRequest) (*UserResponse, *tokens.TokenPairResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*tokens.TokenPairResponse, bool, error)
	LogoutUser(ctx context.Context, accessToken, refreshToken string) error
	GetActiveSessions(ctx context.Context, id string) ([]sessions.SessionResponse, error)
	RevokeSession(ctx context.Context, userID, sessionID string) error
	RevokeAllOtherSessions(ctx context.Context, userID, exceptSessionID string) error
	RevokeAllSessions(ctx context.Context, userID string) error
}

// Implementation

type userService struct {
	userRepo   domains.UserRepository
	tokenSvc   tokens.JWTService
	sessionSvc sessions.SessionService
	validate   *validator.Validate
}

func NewUserService (
	userRepo   domains.UserRepository,
	tokenSvc   tokens.JWTService,
	sessionSvc sessions.SessionService,
	validate   *validator.Validate,
) UserService {
	return &userService{
		userRepo:   userRepo,
		tokenSvc:   tokenSvc,
		sessionSvc: sessionSvc,
		validate:   validate,
	}
}

func (s *userService) RegisterUser(ctx context.Context, req RegisterUserRequest) error {
	if err := jsonValidator.ValidateStruct(s.validate, req); err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return pkg.ErrInternal
	}

	id          := uuid.NewString()
	req.Password = string(hashedPassword)

	payload := ToRegisterUserEntity(id, req)

	if err := s.userRepo.CreateUser(ctx, payload); err != nil {
		return err
	}

	return nil
}

func (s *userService) UpdateUser(ctx context.Context, update *UpdateUserRequest) (*UserResponse, error) {
	if update.RememberMe != nil {
		update.RememberMe = nil
	}

	if err := s.userRepo.UpdateUser(ctx, ToUpdateUserEntity(update)); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetUserByID(ctx, update.ID)
	if err != nil {
		return nil, err
	}

	return ToUserResponse(*user), nil
}

func (s *userService) GetUserByID(ctx context.Context, rawUserID string) (*UserResponse, error) {
    userID, err := parseOrDecodeUUID(rawUserID)
    if err != nil {
        return nil, err
    }

    user, err := s.userRepo.GetUserByID(ctx, userID)
    if err != nil {
        return nil, err
    }

    return ToUserResponse(*user), nil
}

func (s *userService) GetUsers(ctx context.Context) ([]UserResponse, error) {
	users, err := s.userRepo.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	return ToUserListResponse(users), nil
}

func (s *userService) LoginUser(ctx context.Context, agent, ip string, req LoginRequest) (*UserResponse, *tokens.TokenPairResponse, error) {
	if err := jsonValidator.ValidateStruct(s.validate, req); err != nil {
		return nil, nil, err
	}

	hashedPassword, userID, err := s.userRepo.GetPasswordByEmail(ctx, req.Email)
	if err != nil {
		return nil, nil, pkg.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		return nil, nil, pkg.ErrInvalidCredentials
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, pkg.ErrInvalidCredentials
	}

	if err := s.userRepo.UpdateUser(ctx, &domains.UpdateUserEntity{
		ID:         user.ID,
		RememberMe: &req.RememberMe,
	}); err != nil {
		return nil, nil, err
	}

	tokenPair, err := s.tokenSvc.GenerateTokenPair(ctx, user.ID, agent, ip, user.RememberMe)
	if err != nil {
		return nil, nil, err
	}

	user.ID, err = uuidToBase64(user.ID)
	if err != nil {
		return nil, nil, err
	}

	return ToUserResponse(*user), tokens.ToTokenPairResponse(*tokenPair), nil
}

func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (*tokens.TokenPairResponse, bool, error) {
	tokenPair, err := s.tokenSvc.RefreshTokens(ctx, refreshToken)
	if err != nil {
		return nil, false, err
	}

	user, err := s.userRepo.GetUserByID(ctx, tokenPair.UserID)
	if err != nil {
		return nil, false, err
	}

	return tokens.ToTokenPairResponse(*tokenPair), user.RememberMe, nil
}

func (s *userService) LogoutUser(ctx context.Context, accessToken, refreshToken string) error {
	if err := s.tokenSvc.RevokeTokens(ctx, accessToken, refreshToken); err != nil {
		return err
	}

	return nil
}

func (s *userService) GetActiveSessions(ctx context.Context, rawUserID string) ([]sessions.SessionResponse, error) {
    userID, err := parseOrDecodeUUID(rawUserID)
    if err != nil {
        return nil, err
    }

    session, err := s.sessionSvc.GetActiveSessions(ctx, userID)
    if err != nil {
        return nil, err
    }

    res := sessions.ToSessionListResponse(session)
    for i := range session {
        uID, err := uuidToBase64(session[i].UserID)
        if err != nil {
            return nil, err
        }

        sID, err := uuidToBase64(session[i].SessionID)
        if err != nil {
            return nil, err
        }

        res[i].UserID    = uID
        res[i].SessionID = sID
    }

    return res, nil
}

func (s *userService) RevokeSession(ctx context.Context, rawUserID, rawSessionID string) error {
    userID, err := parseOrDecodeUUID(rawUserID)
    if err != nil {
        return err
    }

    sessionID, err := parseOrDecodeUUID(rawSessionID)
    if err != nil {
        return err
    }

    if err := s.sessionSvc.RevokeSession(ctx, userID, sessionID); err != nil {
        return err
    }

    return nil
}

func (s *userService) RevokeAllOtherSessions(ctx context.Context, rawUserID, rawExceptSessionID string) error {
    userID, err := parseOrDecodeUUID(rawUserID)
    if err != nil {
        return err
    }

    exceptSessionID, err := parseOrDecodeUUID(rawExceptSessionID)
    if err != nil {
        return err
    }

    if err := s.sessionSvc.RevokeAllOtherSessions(ctx, userID, exceptSessionID); err != nil {
        return err
    }

    return nil
}

func (s *userService) RevokeAllSessions(ctx context.Context, rawUserID string) error {
    userID, err := parseOrDecodeUUID(rawUserID)
    if err != nil {
        return err
    }

    if err := s.sessionSvc.RevokeAllSessions(ctx, userID); err != nil {
        return err
    }

    return nil
}

// Internal Helper

func uuidToBase64(uuidStr string) (string, error) {
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(parsed[:]), nil
}

func base64ToUUID(encoded string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	parsed, err := uuid.FromBytes(decoded)
	if err != nil {
		return "", err
	}
	return parsed.String(), nil
}

func parseOrDecodeUUID(input string) (string, error) {
    if input == "" {
        return "", pkg.ErrInvalidInput
    }

    if _, err := uuid.Parse(input); err == nil {
        return input, nil
    }

    decoded, err := base64.RawURLEncoding.DecodeString(input)
    if err != nil {
        return "", pkg.ErrInvalidInput
    }

    parsed, err := uuid.FromBytes(decoded)
    if err != nil {
        return "", pkg.ErrInvalidInput
    }

    return parsed.String(), nil
}