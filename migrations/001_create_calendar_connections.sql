-- Migration: Create calendar_connections table for SmartMeet Calendar Integration
-- Phase 1: Calendar Integration (FR1)

-- Create calendar_connections table
CREATE TABLE IF NOT EXISTS calendar_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'google' or 'outlook'
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    token_expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    calendar_email VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Each user can only have one connection per provider
    CONSTRAINT unique_user_provider UNIQUE(user_id, provider)
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_calendar_connections_user_id ON calendar_connections(user_id);
CREATE INDEX IF NOT EXISTS idx_calendar_connections_provider ON calendar_connections(provider);
CREATE INDEX IF NOT EXISTS idx_calendar_connections_active ON calendar_connections(is_active);

-- Add comment for documentation
COMMENT ON TABLE calendar_connections IS 'Stores user calendar provider connections (Google, Outlook) for SmartMeet';
COMMENT ON COLUMN calendar_connections.provider IS 'Calendar provider: google or outlook';
COMMENT ON COLUMN calendar_connections.access_token IS 'OAuth access token for API calls';
COMMENT ON COLUMN calendar_connections.refresh_token IS 'OAuth refresh token to renew access token';
COMMENT ON COLUMN calendar_connections.token_expires_at IS 'When the access token expires';
COMMENT ON COLUMN calendar_connections.calendar_email IS 'Email associated with the calendar';
