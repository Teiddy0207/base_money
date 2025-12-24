package controller

import (
	"go-api-starter/core/config"
	"go-api-starter/core/errors"
	"go-api-starter/core/logger"
	"go-api-starter/core/params"
	"go-api-starter/core/utils"
	"go-api-starter/modules/auth/dto"
	"go-api-starter/modules/auth/validator"
	"net/http"

	"github.com/labstack/echo/v4"
)

func (controller *AuthController) VerifyOTP(c echo.Context) error {
	ctx := c.Request().Context()

	requestData := new(dto.VerifyOTPRequest)
	if err := c.Bind(requestData); err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid request data", nil)
	}

	validationResult := validator.ValidateVerifyOTPRequest(requestData)
	if validationResult.HasError() {
		return controller.BadRequest(errors.ErrInvalidInput, "Invalid request data", validationResult)
	}

	otpResponse, err := controller.AuthService.VerifyOTP(ctx, requestData)
	if err != nil {
		return controller.InternalServerError(errors.ErrInternalServer, "Invalid request data", validationResult)
	}

	return controller.SuccessResponse(c, otpResponse, "Verify OTP success")
}

func (controller *AuthController) ResetPassword(c echo.Context) error {
	ctx := c.Request().Context()

	requestData := new(dto.ResetPasswordRequest)
	if err := c.Bind(requestData); err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid request data", nil)
	}

	validationResult := validator.ValidateResetPasswordRequest(requestData)
	if validationResult.HasError() {
		return controller.BadRequest(errors.ErrInvalidInput, "Invalid request data", validationResult)
	}

	errReset := controller.AuthService.ResetPassword(ctx, requestData)
	if errReset != nil {
		return controller.InternalServerError(errors.ErrInternalServer, "Invalid request data", validationResult)
	}

	return controller.SuccessResponse(c, nil, "Update password success")
}

func (controller *AuthController) Register(c echo.Context) error {
	ctx := c.Request().Context()

	requestData := new(dto.RegisterRequest)
	if err := c.Bind(requestData); err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid request data", nil)
	}

	validationResult := validator.ValidateRegisterRequest(requestData)
	if validationResult.HasError() {
		return controller.BadRequest(errors.ErrInvalidInput, "Invalid request data", validationResult)
	}

	registerResponse, err := controller.AuthService.Register(ctx, requestData)
	if err != nil {
		// Handle different error types appropriately
		if err.Code == errors.ErrAlreadyExists {
			return controller.BadRequest(err.Code, err.Message, nil)
		}
		return controller.InternalServerError(err.Code, err.Message, err)
	}

	return controller.SuccessResponse(c, registerResponse, "Register success")
}

func (controller *AuthController) Login(c echo.Context) error {
	ctx := c.Request().Context()

	requestData := new(dto.LoginRequest)
	if err := c.Bind(requestData); err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid request data", nil)
	}

	validationResult := validator.ValidateLoginRequest(requestData)
	if validationResult.HasError() {
		return controller.BadRequest(errors.ErrInvalidInput, "Invalid request data", validationResult)
	}

	loginResponse, err := controller.AuthService.Login(ctx, requestData)
	if err != nil {
		return controller.InternalServerError(errors.ErrInternalServer, "Invalid request data", err)
	}

	return controller.SuccessResponse(c, loginResponse, "Login success")
}

func (controller *AuthController) Logout(c echo.Context) error {
	ctx := c.Request().Context()

	token, err := utils.GetTokenFromHeader(c)
	if err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid request data", nil)
	}

	errLogout := controller.AuthService.Logout(ctx, token)
	if errLogout != nil {
		logger.Error("AuthController:Logout:Error:", errLogout)
		return controller.InternalServerError(errors.ErrInternalServer, "Logout failed", nil)
	}

	return controller.SuccessResponse(c, nil, "Logout success")
}

func (controller *AuthController) ForgotPassword(c echo.Context) error {

	ctx := c.Request().Context()

	requestData := new(dto.ForgotPasswordRequest)
	if err := c.Bind(requestData); err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid request data", nil)
	}

	validationResult := validator.ValidateForgotPasswordRequest(requestData)
	if validationResult.HasError() {
		return controller.BadRequest(errors.ErrInvalidInput, "Invalid request data", validationResult)
	}

	result, err := controller.AuthService.ForgotPassword(ctx, requestData.Identifier)
	if err != nil {
		return controller.InternalServerError(errors.ErrInternalServer, "Forgot password failed", err)
	}

	return controller.SuccessResponse(c, result, "Forgot password success")
}

func (controller *AuthController) SendOTPChangePassword(c echo.Context) error {
	ctx := c.Request().Context()

	token, err := utils.GetTokenFromHeader(c)
	if err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid token", nil)
	}

	errSend := controller.AuthService.SendOTPChangePassword(ctx, token)
	if errSend != nil {
		return controller.InternalServerError(errors.ErrInternalServer, "Get OTP change password failed", err)
	}

	return controller.SuccessResponse(c, nil, "Get OTP change password success")
}

func (controller *AuthController) ChangePassword(c echo.Context) error {
	ctx := c.Request().Context()

	requestData := new(dto.ChangePasswordRequest)
	if err := c.Bind(requestData); err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid request data", nil)
	}

	validationResult := validator.ValidateChangePasswordRequest(requestData)
	if validationResult.HasError() {
		return controller.BadRequest(errors.ErrInvalidInput, "Invalid request data", validationResult)
	}

	token, err := utils.GetTokenFromHeader(c)
	if err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid token", nil)
	}

	errUpdate := controller.AuthService.ChangePassword(ctx, token, requestData)
	if errUpdate != nil {
		return controller.InternalServerError(errors.ErrInternalServer, "Change password failed", errUpdate)
	}

	return controller.SuccessResponse(c, nil, "Change password success")
}

func (controller *AuthController) RefreshToken(c echo.Context) error {
	ctx := c.Request().Context()

	token, err := utils.GetTokenFromHeader(c)
	if err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid token", nil)
	}

	refreshTokenResponse, errRefresh := controller.AuthService.RefreshToken(ctx, token)
	if errRefresh != nil {
		return controller.InternalServerError(errors.ErrInternalServer, "Refresh token failed", nil)
	}

	return controller.SuccessResponse(c, refreshTokenResponse, "Refresh token success")
}

// GoogleAuth redirects user to Google OAuth login page
func (controller *AuthController) GoogleAuth(c echo.Context) error {
	ctx := c.Request().Context()

	authURL, err := controller.AuthService.GetGoogleAuthURL(ctx)
	if err != nil {
		return controller.InternalServerError(err.Code, err.Message, err)
	}

	return c.Redirect(http.StatusFound, authURL)
}

// GoogleAuthInfo returns OAuth configuration info (for debugging)
func (controller *AuthController) GoogleAuthInfo(c echo.Context) error {
	ctx := c.Request().Context()

	authURL, err := controller.AuthService.GetGoogleAuthURL(ctx)
	if err != nil {
		return controller.InternalServerError(err.Code, err.Message, err)
	}

	cfg, ok := config.GetSafe()
	if !ok {
		return controller.InternalServerError(errors.ErrInternalServer, "config not initialized", nil)
	}

	response := map[string]interface{}{
		"auth_url":     authURL,
		"redirect_uri": cfg.GoogleAPI.RedirectURI,
		"client_id":    cfg.GoogleAPI.ClientID,
		"has_secret":   cfg.GoogleAPI.ClientSecret != "",
		"message":      "Visit the auth_url in your browser to start OAuth flow",
	}

	return controller.SuccessResponse(c, response, "Google OAuth configuration")
}

// GoogleCallback handles the OAuth callback from Google
func (controller *AuthController) GoogleCallback(c echo.Context) error {
	ctx := c.Request().Context()

	code := c.QueryParam("code")
	state := c.QueryParam("state")
	errorParam := c.QueryParam("error")

	// Check if Google returned an error
	if errorParam != "" {
		errorDescription := c.QueryParam("error_description")
		logger.Error("Google OAuth error", "error", errorParam, "description", errorDescription)
		return controller.BadRequest(errors.ErrInvalidRequestData, "Google OAuth error: "+errorParam, nil)
	}

	if code == "" {
		return controller.BadRequest(errors.ErrInvalidRequestData, "authorization code is required. Please initiate OAuth flow by visiting /api/v1/public/auth/google first", nil)
	}

	if state == "" {
		return controller.BadRequest(errors.ErrInvalidRequestData, "state parameter is required for security validation", nil)
	}

	loginResponse, err := controller.AuthService.HandleGoogleCallback(ctx, code, state)
	if err != nil {
		return controller.InternalServerError(err.Code, err.Message, err)
	}

	return controller.SuccessResponse(c, loginResponse, "Google login success")
}

// GetGoogleCalendarEvents retrieves Google Calendar events for authenticated user
func (controller *AuthController) GetGoogleCalendarEvents(c echo.Context) error {
	ctx := c.Request().Context()

	// Get user ID from token
	token, err := utils.GetTokenFromHeader(c)
	if err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid token", nil)
	}

	tokenData, err := utils.ValidateAndParseToken(token)
	if err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid token", nil)
	}

	userID := tokenData.UserID

	// Get query parameters
	queryParams := params.NewQueryParams(c)
	timeMin := c.QueryParam("time_min")
	timeMax := c.QueryParam("time_max")

	events, appErr := controller.AuthService.GetGoogleCalendarEvents(ctx, userID, *queryParams, timeMin, timeMax)
	if appErr != nil {
		return controller.InternalServerError(appErr.Code, appErr.Message, appErr)
	}

	return controller.SuccessResponse(c, events, "Calendar events retrieved successfully")
}

// GetGoogleCalendarList retrieves list of Google Calendars for authenticated user
func (controller *AuthController) GetGoogleCalendarList(c echo.Context) error {
	ctx := c.Request().Context()

	token, err := utils.GetTokenFromHeader(c)
	if err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid token", nil)
	}

	tokenData, err := utils.ValidateAndParseToken(token)
	if err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid token", nil)
	}

	userID := tokenData.UserID
	queryParams := params.NewQueryParams(c)

	calendars, appErr := controller.AuthService.GetGoogleCalendarList(ctx, userID, *queryParams)
	if appErr != nil {
		return controller.InternalServerError(appErr.Code, appErr.Message, appErr)
	}

	return controller.SuccessResponse(c, calendars, "Calendar list retrieved successfully")
}

// GoogleVerify verifies Google idToken and returns access_token and refresh_token
func (controller *AuthController) GoogleVerify(c echo.Context) error {
	ctx := c.Request().Context()

	requestData := new(dto.GoogleVerifyRequest)
	if err := c.Bind(requestData); err != nil {
		return controller.BadRequest(errors.ErrInvalidRequestData, "Invalid request data", nil)
	}

	validationResult := validator.ValidateGoogleVerifyRequest(requestData)
	if validationResult.HasError() {
		return controller.BadRequest(errors.ErrInvalidInput, "Invalid request data", validationResult)
	}

	loginResponse, err := controller.AuthService.VerifyGoogleIdToken(ctx, requestData.IdToken, requestData.AccessToken, requestData.RefreshToken)
	if err != nil {
		return controller.InternalServerError(err.Code, err.Message, err)
	}

	return controller.SuccessResponse(c, loginResponse, "Google login success")
}