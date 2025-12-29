package service

import (
	"context"
	"go-api-starter/modules/auth/repository"
)

// SearchSocialUsers searches for users from social_logins
func (s *AuthService) SearchSocialUsers(ctx context.Context, query string) ([]repository.SocialUserResult, error) {
	if query == "" {
		return s.repo.GetAllSocialUsers(ctx)
	}
	return s.repo.SearchSocialUsers(ctx, query)
}

// GetAllSocialUsers gets all users from social_logins
func (s *AuthService) GetAllSocialUsers(ctx context.Context) ([]repository.SocialUserResult, error) {
	return s.repo.GetAllSocialUsers(ctx)
}
