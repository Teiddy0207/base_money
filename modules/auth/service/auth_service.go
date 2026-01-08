package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-api-starter/core/config"
	"go-api-starter/core/constants"
	"go-api-starter/core/errors"
	"go-api-starter/core/logger"
	"go-api-starter/core/utils"
	"go-api-starter/modules/auth/dto"
	"go-api-starter/modules/auth/entity"
	"go-api-starter/modules/auth/mapper"
	"io"
	"net/http"

	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func (service *AuthService) SendOTPChangePassword(ctx context.Context, token string) *errors.AppError {

	// Check if token is blacklisted
	blacklisted, err := service.cache.IsTokenBlacklisted(ctx, token)
	if err != nil {
		logger.Error("AuthService:SendOTPChangePassword:IsTokenBlacklisted:Error:", err)
		return errors.NewAppError(errors.ErrInternalServer, "failed to check token blacklist", err)
	}
	if blacklisted {
		return errors.NewAppError(errors.ErrUnauthorized, "token is blacklisted", nil)
	}

	tokenData, err := utils.ValidateAndParseToken(token)
	if err != nil {
		logger.Error("AuthService:SendOTPChangePassword:ValidateAndParseToken:Error:", err)
		return errors.NewAppError(errors.ErrInternalServer, "failed to validate token", err)
	}

	user, errGet := service.GetUserByIdentifier(ctx, string(utils.ToString(tokenData.UserID)))
	if errGet != nil || user == nil {
		logger.Error("AuthService:SendOTPChangePassword:GetUserByIdentifier:Error:", errGet)
		return errors.NewAppError(errors.ErrInternalServer, "failed to get user", errGet)
	}

	// Kiểm tra trạng thái xác minh email và phone
	isEmailVerified := user.EmailVerifiedAt != nil
	isPhoneVerified := user.PhoneVerifiedAt != nil

	// Kiểm tra xem có ít nhất một kênh đã được xác minh
	if !isEmailVerified && !isPhoneVerified {
		return errors.NewAppError(errors.ErrUnauthorized, "no verified contact method available", nil)
	}

	// Generate OTP
	otpCode := utils.GenerateOTP()

	// Save OTP to cache
	key := constants.RedisKeyOTPChangePassword + utils.ToString(user.ID)
	errCache := service.cache.SetOTP(ctx, key, otpCode)
	if errCache != nil {
		logger.Error("AuthService:SendOTPChangePassword:SetOTP:Error:", errCache)
		return errors.NewAppError(errors.ErrInternalServer, "failed to save OTP", errCache)
	}

	// Ưu tiên gửi email nếu đã xác minh, nếu không thì gửi SMS
	if isEmailVerified {
		// Tạo template data cho OTP
		data := utils.TemplateData{
			OTPCode: otpCode,
		}

		// Gửi email với OTP template
		errSend := utils.SendTemplateEmailFromTemplatesDir(
			[]string{*user.Email},
			"Your OTP Code",
			"otp_email.html",
			data,
		)
		if errSend != nil {
			logger.Error("AuthService:SendOTPChangePassword:SendTemplateEmailFromTemplatesDir:Error:", errSend)
			return errors.NewAppError(errors.ErrInternalServer, "failed to send OTP email", errSend)
		}
	} else if isPhoneVerified {
		// Gửi SMS OTP
		// TODO: Implement SMS sending functionality
		// errSend := utils.SendSMS(user.Phone, fmt.Sprintf("Your OTP code is: %s", otpCode))
		// if errSend != nil {
		//     logger.Error("AuthService:SendOTPChangePassword:SendSMS:Error:", errSend)
		//     return errors.NewAppError(errors.ErrInternalServer, "failed to send OTP SMS", errSend)
		// }
		logger.Info("SMS OTP sending not implemented yet. OTP code:", otpCode)
	}

	return nil
}

func (service *AuthService) ChangePassword(ctx context.Context, token string, requestData *dto.ChangePasswordRequest) *errors.AppError {

	// Check if token is blacklisted
	blacklisted, err := service.cache.IsTokenBlacklisted(ctx, token)
	if err != nil {
		logger.Error("AuthService:ChangePassword:IsTokenBlacklisted:Error:", err)
		return errors.NewAppError(errors.ErrInternalServer, "failed to check token blacklist", err)
	}
	if blacklisted {
		return errors.NewAppError(errors.ErrUnauthorized, "token is blacklisted", nil)
	}

	parseToken, err := utils.ValidateAndParseToken(token)
	if err != nil {
		logger.Error("AuthService:ChangePassword:ValidateAndParseToken", err)
		return errors.NewAppError(errors.ErrUnauthorized, "invalid token", nil)
	}

	// Check if user exists
	user, errGet := service.GetUserByIdentifier(ctx, utils.ToString(parseToken.UserID))
	if errGet != nil {
		logger.Error("AuthService:ChangePassword:GetUserByIdentifier:Error:", errGet)
		return errors.NewAppError(errors.ErrNotFound, "user not found", errGet)
	}

	// Check if password match
	if !utils.ComparePassword(user.Password, requestData.Password) {
		logger.Error("AuthService:ChangePassword:ComparePassword:Error:", err)
		return errors.NewAppError(errors.ErrUnauthorized, "user has invalid password", nil)
	}

	// Check OTP
	key := constants.RedisKeyOTPChangePassword + utils.ToString(parseToken.UserID)
	otp, err := service.cache.GetOTP(ctx, key)
	if err != nil {
		logger.Error("AuthService:ChangePassword:GetOTP:Error:", err)
		return errors.NewAppError(errors.ErrInternalServer, "failed to get OTP", err)
	}
	if otp != requestData.OTP {
		return errors.NewAppError(errors.ErrUnauthorized, "invalid OTP", nil)
	}

	// Update password
	hashedPassword, err := utils.HashPassword(requestData.NewPassword)
	if err != nil {
		logger.Error("AuthService:ChangePassword:HashPassword:Error:", err)
		return errors.NewAppError(errors.ErrInternalServer, "failed to hash password", err)
	}

	errUpdate := service.repo.PrivateUpdatePasswordUser(ctx, user.ID, hashedPassword)
	if errUpdate != nil {
		logger.Error("AuthService:ChangePassword:UpdateUser:Error:", errUpdate)
		return errors.NewAppError(errors.ErrInternalServer, "failed to change password", errUpdate)
	}
	// Invalid token
	errAdd := service.cache.AddToTokenBlacklist(ctx, token)
	if errAdd != nil {
		logger.Error("AuthService:ChangePassword:AddToBlacklist:Error:", errAdd)
		return errors.NewAppError(errors.ErrInternalServer, "failed to add token to blacklist", errAdd)
	}

	return nil
}

func (service *AuthService) ForgotPassword(ctx context.Context, identifier string) (*dto.ForgotPasswordResponse, *errors.AppError) {

	// Check if user exists
	user, err := service.GetUserByIdentifier(ctx, identifier)
	if err != nil || user == nil {
		logger.Error("AuthService:ForgotPassword:GetUserByIdentifier:Error:", err)
		return nil, errors.NewAppError(errors.ErrNotFound, "user not found", nil)
	}

	identifierType := utils.DetectIdentifierType(identifier)
	if identifierType == utils.IdentifierTypeUnknown {
		return nil, errors.NewAppError(errors.ErrUnauthorized, "invalid identifier", nil)
	}

	if identifierType == utils.IdentifierTypeEmail {
		// Check if user has verified their email
		if user.EmailVerifiedAt == nil {
			return nil, errors.NewAppError(errors.ErrUnauthorized, "user not verified email", nil)
		}

		otpCode := utils.GenerateOTP()

		// Tạo template data cho OTP
		data := utils.TemplateData{
			OTPCode: otpCode,
		}

		// Save OTP to cache
		errCache := service.cache.SetOTP(ctx, utils.ToString(user.ID), otpCode)
		if errCache != nil {
			logger.Error("AuthService:ForgotPassword:SetOTP:Error:", errCache)
			return nil, errors.NewAppError(errors.ErrInternalServer, "failed to save OTP", errCache)
		}

		// Gửi email với OTP template
		errSend := utils.SendTemplateEmailFromTemplatesDir(
			[]string{*user.Email},
			"Your OTP Code",
			"otp_email.html",
			data,
		)
		if errSend != nil {
			logger.Error("AuthService:ForgotPassword:SendTemplateEmailFromTemplatesDir:Error:", errSend)
			return nil, errors.NewAppError(errors.ErrInternalServer, "failed to send OTP email", errSend)
		}

	}

	if identifierType == utils.IdentifierTypePhone {
		if user.PhoneVerifiedAt == nil {
			return nil, errors.NewAppError(errors.ErrUnauthorized, "user not verified phone", nil)
		}
		// TODO: Implement SMS OTP sending
	}

	return &dto.ForgotPasswordResponse{
		UserId: user.ID,
	}, nil
}

func (service *AuthService) VerifyOTP(ctx context.Context, requestData *dto.VerifyOTPRequest) (*dto.VerifyOTPResponse, *errors.AppError) {
	// Get OTP from cache
	otp, err := service.cache.GetOTP(ctx, utils.ToString(requestData.UserID))
	if err != nil || otp == "" {
		logger.Error("AuthService:VerifyOTP:GetOTP:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to get OTP from cache", err)
	}

	// Compare OTP
	if otp != requestData.OTP {
		return nil, errors.NewAppError(errors.ErrUnauthorized, "invalid OTP", nil)
	}

	resetPasswordToken, err := utils.GenerateToken(requestData.UserID, nil, nil, constants.ScopeTokenResetPassword)
	if err != nil {
		logger.Error("AuthService:VerifyOTP:GenerateToken:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to generate token", err)
	}

	return &dto.VerifyOTPResponse{
		Token: resetPasswordToken,
	}, nil

}

func (service *AuthService) ResetPassword(ctx context.Context, requestData *dto.ResetPasswordRequest) *errors.AppError {

	// Check if token is blacklisted
	blacklisted, err := service.cache.IsTokenBlacklisted(ctx, requestData.Token)
	if err != nil {
		logger.Error("AuthService:ResetPassword:IsTokenBlacklisted:Error:", err)
		return errors.NewAppError(errors.ErrInternalServer, "failed to check token blacklist", err)
	}
	if blacklisted {
		return errors.NewAppError(errors.ErrUnauthorized, "token is blacklisted", nil)
	}

	tokenData, err := utils.ValidateAndParseToken(requestData.Token)
	if err != nil {
		logger.Error("AuthService:ResetPassword:ValidateAndParseToken:Error:", err)
		return errors.NewAppError(errors.ErrUnauthorized, "invalid token", nil)
	}

	if tokenData.Scope != constants.ScopeTokenResetPassword {
		return errors.NewAppError(errors.ErrUnauthorized, "invalid token", nil)
	}

	hashPassword, err := utils.HashPassword(requestData.NewPassword)
	if err != nil {
		logger.Error("AuthService:ResetPassword:HashPassword:Error:", err)
		return errors.NewAppError(errors.ErrInternalServer, "failed to hash password", err)
	}

	errUpdate := service.repo.PrivateUpdatePasswordUser(ctx, tokenData.UserID, hashPassword)
	if errUpdate != nil {
		logger.Error("AuthService:ResetPassword:UpdateUser:Error:", errUpdate)
		return errors.NewAppError(errors.ErrInternalServer, "failed to update user", errUpdate)
	}

	// Add token to blacklist
	errBlacklist := service.cache.AddToTokenBlacklist(ctx, requestData.Token)
	if errBlacklist != nil {
		logger.Error("AuthService:ResetPassword:AddToBlacklist:Error:", errBlacklist)
		return errors.NewAppError(errors.ErrInternalServer, "failed to add token to blacklist", errBlacklist)
	}

	return nil
}

func (service *AuthService) Logout(ctx context.Context, token string) *errors.AppError {
	// Add token to blacklist
	err := service.cache.AddToTokenBlacklist(ctx, token)
	if err != nil {
		logger.Error("AuthService:Logout:AddToBlacklist:Error:", err)
		return errors.NewAppError(errors.ErrInternalServer, "failed to add token to blacklist", err)
	}
	return nil
}

// Login authenticates a user with their identifier (phone/email) and password
// It implements login attempt blocking to prevent brute force attacks
func (service *AuthService) Login(ctx context.Context, requestData *dto.LoginRequest) (*dto.LoginResponse, *errors.AppError) {
	// Create unique key for tracking login attempts per user
	loginKey := fmt.Sprintf("login:%s", requestData.Identifier)

	// Check if user is currently blocked due to too many failed login attempts
	loginCount, err := service.cache.IsLoginBlocked(ctx, loginKey)
	if err != nil {
		logger.Error("AuthService:Login:IsLoginBlocked:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to get login attempt", err)
	}

	// If user is blocked, refresh the block duration and return error
	if loginCount {
		errExpire := service.cache.Expire(ctx, loginKey, constants.BlockDuration)
		if errExpire != nil {
			logger.Error("AuthService:Login:Expire:Error:", errExpire)
			return nil, errors.NewAppError(errors.ErrInternalServer, "failed to expire login attempt", err)
		}
		return nil, errors.NewAppError(errors.ErrUnauthorized, "user is locked in 15 minite", nil)
	}

	// Retrieve user using identifier (phone or email)
	user, errGet := service.GetUserByIdentifier(ctx, requestData.Identifier)
	if errGet != nil || user == nil {
		return nil, errors.NewAppError(errors.ErrNotFound, "user not found", nil)
	}

	// Check if identifier is email or phone
	identifierType := utils.DetectIdentifierType(requestData.Identifier)
	if identifierType == utils.IdentifierTypeUnknown {
		return nil, errors.NewAppError(errors.ErrUnauthorized, "invalid identifier", nil)
	}

	if identifierType == utils.IdentifierTypeEmail {
		// Check if user has verified their email
		if user.EmailVerifiedAt == nil {
			errIncrement := service.cache.IncrementLoginAttempt(ctx, loginKey)
			if errIncrement != nil {
				logger.Error("AuthService:Login:IncrementLoginAttempt:Error:", errIncrement)
				return nil, errors.NewAppError(errors.ErrInternalServer, "failed to increment login attempt", err)
			}
			return nil, errors.NewAppError(errors.ErrUnauthorized, "user not verified email", nil)
		}
	}

	if identifierType == utils.IdentifierTypePhone {
		// Check if user has verified their phone
		if user.PhoneVerifiedAt == nil {
			errIncrement := service.cache.IncrementLoginAttempt(ctx, loginKey)
			if errIncrement != nil {
				logger.Error("AuthService:Login:IncrementLoginAttempt:Error:", errIncrement)
				return nil, errors.NewAppError(errors.ErrInternalServer, "failed to increment login attempt", err)
			}
			return nil, errors.NewAppError(errors.ErrUnauthorized, "user not verified phone", nil)
		}
	}

	// Check if user account is active
	if !user.IsActive {
		errIncrement := service.cache.IncrementLoginAttempt(ctx, loginKey)
		if errIncrement != nil {
			logger.Error("AuthService:Login:IncrementLoginAttempt:Error:", errIncrement)
			return nil, errors.NewAppError(errors.ErrInternalServer, "failed to increment login attempt", err)
		}
		return nil, errors.NewAppError(errors.ErrUnauthorized, "user not active", nil)
	}

	// Verify password - if incorrect, increment failed login attempts
	if !utils.ComparePassword(user.Password, requestData.Password) {
		//Increment failed login attempt counter
		errIncrement := service.cache.IncrementLoginAttempt(ctx, loginKey)
		if errIncrement != nil {
			logger.Error("AuthService:Login:IncrementLoginAttempt:Error:", errIncrement)
			return nil, errors.NewAppError(errors.ErrInternalServer, "failed to increment login attempt", err)
		}
		logger.Error("AuthService:Login:IncrementLoginAttempt:Error:", errIncrement)
		return nil, errors.NewAppError(errors.ErrUnauthorized, "incorrect password", nil)
	}

	// Generate JWT access token for API authentication
	accessToken, err := utils.GenerateToken(user.ID, user.Email, user.Username, constants.ScopeTokenAccess)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to generate access token", err)
	}

	// Generate JWT refresh token for token renewal
	refreshToken, err := utils.GenerateToken(user.ID, user.Email, user.Username, constants.ScopeTokenRefresh)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to generate refresh token", err)
	}

	// Clear any existing login attempts for this user
	errExpire := service.cache.Del(ctx, loginKey)
	if errExpire != nil {
		logger.Error("AuthService:Login:Expire:Error:", errExpire)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to expire login attempt", err)
	}

	// Return successful login response with both tokens
	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (service *AuthService) GetUserByIdentifier(ctx context.Context, identifier string) (*dto.UserResponse, *errors.AppError) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultTimeout)
	defer cancel()

	// TODO: Implement cache user info by identifier
	result, err := service.repo.GetUserByIdentifier(ctx, identifier)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to get user by identifier", err)
	}

	return mapper.ToUserDTO(result), nil
}

func (service *AuthService) Register(ctx context.Context, requestData *dto.RegisterRequest) (*dto.RegisterResponse, *errors.AppError) {
	// Check if user already exists
	existingUser, _ := service.GetUserByIdentifier(ctx, requestData.Phone)
	if existingUser != nil {
		return nil, errors.NewAppError(errors.ErrAlreadyExists, "user with phone already exists", nil)
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(requestData.Password)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to hash password", err)
	}

	// Create user entity
	userEntity := &entity.User{
		Phone:    requestData.Phone,
		Password: hashedPassword,
		IsActive: true, // Set default to true for new users
	}

	// Save user to database
	createdUser, err := service.repo.CreateUser(ctx, userEntity)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to create user", err)
	}

	// Generate JWT tokens
	accessToken, err := utils.GenerateToken(createdUser.ID, createdUser.Email, createdUser.Username, constants.ScopeTokenAccess)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to generate access token", err)
	}

	refreshToken, err := utils.GenerateToken(createdUser.ID, createdUser.Email, createdUser.Username, constants.ScopeTokenRefresh)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to generate refresh token", err)
	}

	return &dto.RegisterResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (service *AuthService) RefreshToken(ctx context.Context, token string) (*dto.RefreshTokenResponse, *errors.AppError) {
	// TODO: Check if token is blacklisted
	isBlacklisted, errCheck := service.cache.IsTokenBlacklisted(ctx, token)
	if errCheck != nil {
		return nil, errors.NewAppError(errors.ErrUnauthorized, "failed to check token", nil)
	}
	if isBlacklisted {
		return nil, errors.NewAppError(errors.ErrUnauthorized, "token is blacklisted", nil)
	}

	user, err := utils.ValidateAndParseToken(token)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrUnauthorized, "failed to parse token", nil)
	}

	result, err := service.repo.GetUserByIdentifier(ctx, user.ID)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to get user by identifier", err)
	}

	// Generate new tokens
	accessToken, err := utils.GenerateToken(result.ID, result.Email, nil, constants.ScopeTokenAccess)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to generate access token", err)
	}

	refreshToken, err := utils.GenerateToken(result.ID, result.Email, nil, constants.ScopeTokenRefresh)
	if err != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to generate refresh token", err)
	}

	// Add Refresh Token to Blacklist
	errAdd := service.cache.AddToTokenBlacklist(ctx, refreshToken)
	if errAdd != nil {
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to add refresh token to blacklist", errAdd)
	}

	// Return new tokens
	return &dto.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// GetGoogleAuthURL generates the Google OAuth authorization URL
func (service *AuthService) GetGoogleAuthURL(ctx context.Context) (string, *errors.AppError) {
	cfg, ok := config.GetSafe()
	if !ok {
		return "", errors.NewAppError(errors.ErrInternalServer, "config not initialized", nil)
	}

	if cfg.GoogleAPI.ClientID == "" || cfg.GoogleAPI.ClientSecret == "" || cfg.GoogleAPI.RedirectURI == "" {
		return "", errors.NewAppError(errors.ErrInternalServer, "Google OAuth configuration is missing", nil)
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleAPI.ClientID,
		ClientSecret: cfg.GoogleAPI.ClientSecret,
		RedirectURL:  cfg.GoogleAPI.RedirectURI,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/calendar.readonly", // Google Calendar read access
		},
		Endpoint: google.Endpoint,
	}

	// Generate state token for CSRF protection
	state := utils.GenerateRandomString(32)

	// Store state in database for validation (10 minutes expiry)
	expiresAt := time.Now().Add(10 * time.Minute)
	err := service.repo.SaveOAuthState(ctx, state, expiresAt)
	if err != nil {
		logger.Error("AuthService:GetGoogleAuthURL:SaveOAuthState:Error", "error", err, "state", state)
		return "", errors.NewAppError(errors.ErrInternalServer, "failed to store state token in database", err)
	}

	logger.Info("AuthService:GetGoogleAuthURL:StateStored", "state", state, "expires_at", expiresAt)

	authURL := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	return authURL, nil
}

// HandleGoogleCallback handles the OAuth callback from Google
func (service *AuthService) HandleGoogleCallback(ctx context.Context, code string, state string) (*dto.LoginResponse, *errors.AppError) {
	// Validate state token from database
	oauthState, err := service.repo.GetOAuthState(ctx, state)
	if err != nil {
		logger.Error("AuthService:HandleGoogleCallback:GetOAuthState:Error", "error", err, "state", state)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to validate state token", err)
	}

	if oauthState == nil {
		logger.Error("AuthService:HandleGoogleCallback:StateNotFound", "state", state)
		return nil, errors.NewAppError(errors.ErrUnauthorized, "invalid or expired state token. Please initiate OAuth flow again by visiting /api/v1/public/auth/google", nil)
	}

	// Delete state token after use (one-time use)
	err = service.repo.DeleteOAuthState(ctx, state)
	if err != nil {
		logger.Error("AuthService:HandleGoogleCallback:DeleteOAuthState:Error", "error", err, "state", state)
		// Continue even if delete fails
	}

	cfg, ok := config.GetSafe()
	if !ok {
		return nil, errors.NewAppError(errors.ErrInternalServer, "config not initialized", nil)
	}

	if cfg.GoogleAPI.ClientID == "" || cfg.GoogleAPI.ClientSecret == "" || cfg.GoogleAPI.RedirectURI == "" {
		return nil, errors.NewAppError(errors.ErrInternalServer, "Google OAuth configuration is missing", nil)
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleAPI.ClientID,
		ClientSecret: cfg.GoogleAPI.ClientSecret,
		RedirectURL:  cfg.GoogleAPI.RedirectURI,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/calendar.readonly", // Google Calendar read access
		},
		Endpoint: google.Endpoint,
	}

	// Exchange authorization code for token
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		logger.Error("AuthService:HandleGoogleCallback:Exchange:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to exchange token", err)
	}

	// Get user info from Google
	userInfo, err := service.getGoogleUserInfo(ctx, token.AccessToken)
	if err != nil {
		logger.Error("AuthService:HandleGoogleCallback:GetGoogleUserInfo:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to get user info", err)
	}

	// Find or create user
	user, errGet := service.repo.GetUserByIdentifier(ctx, userInfo.Email)
	if errGet != nil {
		logger.Error("AuthService:HandleGoogleCallback:GetUserByIdentifier:Error:", errGet)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to get user", errGet)
	}

	if user == nil {
		hashedPassword, _ := utils.HashPassword(utils.GenerateRandomString(32))
		username := userInfo.Name
		newUser := &entity.User{
			Email:    &userInfo.Email,
			Phone:    "", // Empty string for Google login users
			Username: &username,
			Password: hashedPassword,
			IsActive: true, // Set default to true for new users
		}

		// Mark email as verified if Google says it's verified
		if userInfo.VerifiedEmail {
			now := time.Now()
			newUser.EmailVerifiedAt = &now
		}

		createdUser, errCreate := service.repo.CreateUser(ctx, newUser)
		if errCreate != nil {
			logger.Error("AuthService:HandleGoogleCallback:CreateUser:Error:", errCreate)
			return nil, errors.NewAppError(errors.ErrInternalServer, "failed to create user", errCreate)
		}
		user = createdUser
	}

	provider, err := service.repo.GetOAuthProviderByName(ctx, "google")
	if err != nil {
		logger.Error("AuthService:HandleGoogleCallback:GetOAuthProviderByName:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to get Google provider", err)
	}
	if provider == nil {
		logger.Error("AuthService:HandleGoogleCallback:ProviderNotFound", "provider", "google")
		return nil, errors.NewAppError(errors.ErrInternalServer, "Google provider not found in database", nil)
	}

	providerUserID := uuid.New()
	if userInfo.ID != "" {
		hashed := uuid.NewSHA1(uuid.NameSpaceOID, []byte("google:"+userInfo.ID))
		providerUserID = hashed
	}
	providerUsername := userInfo.Name
	providerEmail := userInfo.Email
	expiresAt := token.Expiry
	now := time.Now()

	socialLogin := &entity.SocialLogin{
		UserID:           user.ID,
		ProviderID:       provider.ID,
		ProviderUserID:   providerUserID,
		ProviderUsername: &providerUsername,
		ProviderEmail:    &providerEmail,
		AccessToken:      &token.AccessToken,
		RefreshToken:     &token.RefreshToken,
		TokenExpiresAt:   &expiresAt,
		LastLoginAt:      &now,
		IsActive:         true,
	}

	if err := service.repo.SaveOrUpdateSocialLogin(ctx, socialLogin); err != nil {
		logger.Error("AuthService:HandleGoogleCallback:SaveOrUpdateSocialLogin:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to save Google tokens", err)
	}

	logger.Info("AuthService:HandleGoogleCallback:GoogleTokensSaved",
		"user_id", user.ID,
		"has_access_token", token.AccessToken != "",
		"has_refresh_token", token.RefreshToken != "",
		"expires_at", expiresAt)

	// Generate JWT tokens
	accessToken, err := utils.GenerateToken(user.ID, user.Email, user.Username, constants.ScopeTokenAccess)
	if err != nil {
		logger.Error("AuthService:HandleGoogleCallback:GenerateAccessToken:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to generate access token", err)
	}

	refreshToken, err := utils.GenerateToken(user.ID, user.Email, user.Username, constants.ScopeTokenRefresh)
	if err != nil {
		logger.Error("AuthService:HandleGoogleCallback:GenerateRefreshToken:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to generate refresh token", err)
	}

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// getGoogleUserInfo fetches user information from Google API
func (service *AuthService) getGoogleUserInfo(ctx context.Context, accessToken string) (*GoogleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// GoogleUserInfo represents Google user information
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// GoogleTokenInfo represents the response from Google's tokeninfo API
type GoogleTokenInfo struct {
	Iss           string `json:"iss"`
	Azp           string `json:"azp"`
	Aud           string `json:"aud"`
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Iat           string `json:"iat"`
	Exp           string `json:"exp"`
	Alg           string `json:"alg"`
	Kid           string `json:"kid"`
}

// VerifyGoogleIdToken verifies Google idToken and returns login response
// googleAccessToken và googleRefreshToken là optional - nếu có sẽ được lưu để gọi Google APIs như Calendar
// serverAuthCode là optional - nếu có, backend sẽ exchange để lấy refresh_token
func (service *AuthService) VerifyGoogleIdToken(ctx context.Context, idToken string, googleAccessToken string, googleRefreshToken string, serverAuthCode string) (*dto.LoginResponse, *errors.AppError) {
	cfg, ok := config.GetSafe()
	if !ok {
		return nil, errors.NewAppError(errors.ErrInternalServer, "config not initialized", nil)
	}

	if cfg.GoogleAPI.ClientID == "" {
		return nil, errors.NewAppError(errors.ErrInternalServer, "Google OAuth configuration is missing", nil)
	}

	// If serverAuthCode is provided and we don't have refresh_token, exchange it
	if serverAuthCode != "" && googleRefreshToken == "" {
		logger.Info("AuthService:VerifyGoogleIdToken:ExchangingServerAuthCode")
		tokens, err := service.exchangeAuthCodeForTokens(ctx, serverAuthCode, cfg)
		if err != nil {
			logger.Error("AuthService:VerifyGoogleIdToken:ExchangeAuthCode:Error:", err)
			// Don't fail login, just log the error
		} else {
			if tokens.AccessToken != "" {
				googleAccessToken = tokens.AccessToken
			}
			if tokens.RefreshToken != "" {
				googleRefreshToken = tokens.RefreshToken
				logger.Info("AuthService:VerifyGoogleIdToken:RefreshTokenObtained")
			}
		}
	}

	// Verify token using Google's tokeninfo API
	tokenInfo, err := service.verifyGoogleTokenInfo(ctx, idToken, cfg.GoogleAPI.ClientID)
	if err != nil {
		logger.Error("AuthService:VerifyGoogleIdToken:VerifyGoogleTokenInfo:Error:", err)
		return nil, errors.NewAppError(errors.ErrUnauthorized, "invalid Google idToken", err)
	}

	// Convert token info to GoogleUserInfo format
	userInfo := &GoogleUserInfo{
		ID:            tokenInfo.Sub,
		Email:         tokenInfo.Email,
		VerifiedEmail: tokenInfo.EmailVerified == "true",
		Name:          tokenInfo.Name,
		GivenName:     tokenInfo.GivenName,
		FamilyName:    tokenInfo.FamilyName,
		Picture:       tokenInfo.Picture,
	}

	// Find or create user
	user, errGet := service.repo.GetUserByIdentifier(ctx, userInfo.Email)
	if errGet != nil {
		logger.Error("AuthService:VerifyGoogleIdToken:GetUserByIdentifier:Error:", errGet)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to get user", errGet)
	}

	if user == nil {
		hashedPassword, _ := utils.HashPassword(utils.GenerateRandomString(32))
		username := userInfo.Name
		if username == "" {
			username = userInfo.Email
		}
		newUser := &entity.User{
			Email:    &userInfo.Email,
			Phone:    "", // Empty string for Google login users
			Username: &username,
			Password: hashedPassword,
			IsActive: true, // Set default to true for new users
		}

		// Mark email as verified if Google says it's verified
		if userInfo.VerifiedEmail {
			now := time.Now()
			newUser.EmailVerifiedAt = &now
		}

		createdUser, errCreate := service.repo.CreateUser(ctx, newUser)
		if errCreate != nil {
			logger.Error("AuthService:VerifyGoogleIdToken:CreateUser:Error:", errCreate)
			return nil, errors.NewAppError(errors.ErrInternalServer, "failed to create user", errCreate)
		}
		user = createdUser
	}

	// Get or create Google provider
	provider, err := service.repo.GetOAuthProviderByName(ctx, "google")
	if err != nil {
		logger.Error("AuthService:VerifyGoogleIdToken:GetOAuthProviderByName:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to get Google provider", err)
	}
	if provider == nil {
		logger.Error("AuthService:VerifyGoogleIdToken:ProviderNotFound", "provider", "google")
		return nil, errors.NewAppError(errors.ErrInternalServer, "Google provider not found in database", nil)
	}

	// Save or update social login với access token nếu có
	providerUserID := uuid.New()
	if userInfo.ID != "" {
		hashed := uuid.NewSHA1(uuid.NameSpaceOID, []byte("google:"+userInfo.ID))
		providerUserID = hashed
	}
	providerUsername := userInfo.Name
	providerEmail := userInfo.Email
	now := time.Now()

	socialLogin := &entity.SocialLogin{
		UserID:           user.ID,
		ProviderID:       provider.ID,
		ProviderUserID:   providerUserID,
		ProviderUsername: &providerUsername,
		ProviderEmail:    &providerEmail,
		LastLoginAt:      &now,
		IsActive:         true,
	}

	// Lưu Google access token và refresh token nếu có (để gọi Google APIs như Calendar)
	if googleAccessToken != "" {
		socialLogin.AccessToken = &googleAccessToken
		// Google access token thường hết hạn sau 1 giờ, set mặc định
		expiresAt := now.Add(1 * time.Hour)
		socialLogin.TokenExpiresAt = &expiresAt
		logger.Info("AuthService:VerifyGoogleIdToken:GoogleAccessTokenSaved",
			"user_id", user.ID,
			"has_access_token", true,
			"expires_at", expiresAt)
	}

	if googleRefreshToken != "" {
		socialLogin.RefreshToken = &googleRefreshToken
		logger.Info("AuthService:VerifyGoogleIdToken:GoogleRefreshTokenSaved",
			"user_id", user.ID,
			"has_refresh_token", true)
	}

	if err := service.repo.SaveOrUpdateSocialLogin(ctx, socialLogin); err != nil {
		logger.Error("AuthService:VerifyGoogleIdToken:SaveOrUpdateSocialLogin:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to save Google login", err)
	}

	// Generate JWT tokens
	accessToken, err := utils.GenerateToken(user.ID, user.Email, user.Username, constants.ScopeTokenAccess)
	if err != nil {
		logger.Error("AuthService:VerifyGoogleIdToken:GenerateAccessToken:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to generate access token", err)
	}

	refreshToken, err := utils.GenerateToken(user.ID, user.Email, user.Username, constants.ScopeTokenRefresh)
	if err != nil {
		logger.Error("AuthService:VerifyGoogleIdToken:GenerateRefreshToken:Error:", err)
		return nil, errors.NewAppError(errors.ErrInternalServer, "failed to generate refresh token", err)
	}

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// verifyGoogleTokenInfo verifies Google idToken using tokeninfo API
func (service *AuthService) verifyGoogleTokenInfo(ctx context.Context, idToken, clientID string) (*GoogleTokenInfo, error) {
	url := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to verify token: %s", string(body))
	}

	var tokenInfo GoogleTokenInfo
	if err := json.Unmarshal(body, &tokenInfo); err != nil {
		return nil, err
	}

	// Verify audience matches client ID
	if tokenInfo.Aud != clientID {
		return nil, fmt.Errorf("token audience does not match client ID")
	}

	// Verify issuer
	if tokenInfo.Iss != "https://accounts.google.com" && tokenInfo.Iss != "accounts.google.com" {
		return nil, fmt.Errorf("invalid token issuer")
	}

	return &tokenInfo, nil
}

// exchangeAuthCodeForTokens exchanges serverAuthCode for access_token and refresh_token
func (service *AuthService) exchangeAuthCodeForTokens(ctx context.Context, serverAuthCode string, cfg *config.Config) (*GoogleToken, error) {
	// Use oauth2 library to exchange code
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.GoogleAPI.ClientID,
		ClientSecret: cfg.GoogleAPI.ClientSecret,
		RedirectURL:  "", // Empty for mobile apps
		Scopes: []string{
			"https://www.googleapis.com/auth/calendar",
			"https://www.googleapis.com/auth/calendar.events",
		},
		Endpoint: google.Endpoint,
	}

	// Exchange the code
	token, err := oauth2Config.Exchange(ctx, serverAuthCode)
	if err != nil {
		logger.Error("exchangeAuthCodeForTokens:Exchange:Error:", err)
		return nil, err
	}

	result := &GoogleToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
	}

	logger.Info("exchangeAuthCodeForTokens:Success",
		"has_access_token", result.AccessToken != "",
		"has_refresh_token", result.RefreshToken != "",
		"expires_at", result.ExpiresAt)

	return result, nil
}
