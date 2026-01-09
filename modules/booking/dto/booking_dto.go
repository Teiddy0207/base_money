package dto

// PersonalBookingURLResponse represents the response for personal booking URL
type PersonalBookingURLResponse struct {
	URL string `json:"url"`
}

// WeekStatisticsResponse represents weekly event statistics
type WeekStatisticsResponse struct {
	TotalEvents      int    `json:"total_events"`       // Tổng số sự kiện tuần này
	TotalDurationMinutes int `json:"total_duration_minutes"` // Tổng thời gian (phút)
	TotalDurationHours   float64 `json:"total_duration_hours"` // Tổng thời gian (giờ) - làm tròn 2 chữ số
}

