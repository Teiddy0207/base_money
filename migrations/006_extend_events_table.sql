-- FR2: Smart Scheduler - Extend events table
-- Created: 2024-12-29

-- Add new columns to events table for smart scheduling
ALTER TABLE events ADD COLUMN IF NOT EXISTS host_id UUID REFERENCES users(id);
ALTER TABLE events ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE events ADD COLUMN IF NOT EXISTS duration_minutes INTEGER DEFAULT 60;
ALTER TABLE events ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'pending';
ALTER TABLE events ADD COLUMN IF NOT EXISTS timezone VARCHAR(50) DEFAULT 'Asia/Ho_Chi_Minh';
ALTER TABLE events ADD COLUMN IF NOT EXISTS meeting_link TEXT;
ALTER TABLE events ADD COLUMN IF NOT EXISTS preferences JSONB DEFAULT '{}';

-- Rename start_date/end_date to scheduled_start/scheduled_end for clarity (optional, keep both for compatibility)
-- ALTER TABLE events RENAME COLUMN start_date TO scheduled_start;
-- ALTER TABLE events RENAME COLUMN end_date TO scheduled_end;

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_events_host_id ON events(host_id);
CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);

-- Add has_calendar_connected column to user_events for tracking
ALTER TABLE user_events ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'pending';
ALTER TABLE user_events ADD COLUMN IF NOT EXISTS has_calendar_connected BOOLEAN DEFAULT false;

-- Create event_slots table for suggested time slots
CREATE TABLE IF NOT EXISTS event_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    available_count INTEGER DEFAULT 0,
    total_participants INTEGER DEFAULT 0,
    score INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_event_slots_event_id ON event_slots(event_id);
