package service

import (
	"go-api-starter/modules/meeting/entity"
	"sort"
	"time"
)

// SlotFinder handles the algorithm to find available time slots
type SlotFinder struct {
	// BusinessHoursStart - default 8:00
	BusinessHoursStart int
	// BusinessHoursEnd - default 18:00
	BusinessHoursEnd int
	// SlotDurationMinutes for slot generation
	SlotDurationMinutes int
}

// NewSlotFinder creates a new slot finder with default settings
func NewSlotFinder() *SlotFinder {
	return &SlotFinder{
		BusinessHoursStart:  8,
		BusinessHoursEnd:    18,
		SlotDurationMinutes: 30,
	}
}

// FindAvailableSlots finds common free slots across all participants
func (sf *SlotFinder) FindAvailableSlots(
	eventDuration int,
	searchStart time.Time,
	searchEnd time.Time,
	busyTimes []entity.TimeSlot,
	preferences *entity.EventPreferences,
	totalParticipants int,
) []entity.EventSlot {

	// 1. Merge overlapping busy times
	mergedBusy := sf.mergeOverlappingSlots(busyTimes)

	// 2. Generate possible slots
	allSlots := sf.generateTimeSlots(searchStart, searchEnd, eventDuration, preferences)

	// 3. Filter out busy slots
	freeSlots := sf.filterBusySlots(allSlots, mergedBusy)

	// 4. Apply preferences and score
	scoredSlots := sf.scoreSlots(freeSlots, preferences, totalParticipants)

	// 5. Sort by score (descending)
	sort.Slice(scoredSlots, func(i, j int) bool {
		return scoredSlots[i].Score > scoredSlots[j].Score
	})

	// 6. Return top 10 slots
	if len(scoredSlots) > 10 {
		return scoredSlots[:10]
	}
	return scoredSlots
}

// mergeOverlappingSlots merges overlapping busy time slots
func (sf *SlotFinder) mergeOverlappingSlots(slots []entity.TimeSlot) []entity.TimeSlot {
	if len(slots) == 0 {
		return slots
	}

	// Sort by start time
	sort.Slice(slots, func(i, j int) bool {
		return slots[i].Start.Before(slots[j].Start)
	})

	merged := []entity.TimeSlot{slots[0]}

	for i := 1; i < len(slots); i++ {
		last := &merged[len(merged)-1]
		current := slots[i]

		// If overlapping or adjacent, extend
		if current.Start.Before(last.End) || current.Start.Equal(last.End) {
			if current.End.After(last.End) {
				last.End = current.End
			}
		} else {
			merged = append(merged, current)
		}
	}

	return merged
}

// generateTimeSlots generates potential time slots within the search range
func (sf *SlotFinder) generateTimeSlots(
	start time.Time,
	end time.Time,
	durationMinutes int,
	preferences *entity.EventPreferences,
) []entity.TimeSlot {

	slots := []entity.TimeSlot{}
	duration := time.Duration(durationMinutes) * time.Minute

	// Round start to next half hour
	current := sf.roundToNextHalfHour(start)

	for current.Add(duration).Before(end) || current.Add(duration).Equal(end) {
		// Check if within business hours
		hour := current.Hour()
		slotEndHour := current.Add(duration).Hour()

		// Apply business hours filter
		if preferences != nil && preferences.OnlyBusinessHours {
			if hour < sf.BusinessHoursStart || slotEndHour > sf.BusinessHoursEnd {
				current = current.Add(30 * time.Minute)
				continue
			}
		}

		// Apply weekend filter
		if preferences != nil && preferences.ExcludeWeekends {
			weekday := current.Weekday()
			if weekday == time.Saturday || weekday == time.Sunday {
				// Skip to Monday
				daysToMonday := (8 - int(weekday)) % 7
				if daysToMonday == 0 {
					daysToMonday = 1
				}
				current = time.Date(current.Year(), current.Month(), current.Day()+daysToMonday,
					sf.BusinessHoursStart, 0, 0, 0, current.Location())
				continue
			}
		}

		// Only generate slots at 00 or 30 minutes
		minute := current.Minute()
		if minute != 0 && minute != 30 {
			current = sf.roundToNextHalfHour(current)
			continue
		}

		slot := entity.TimeSlot{
			Start: current,
			End:   current.Add(duration),
		}
		slots = append(slots, slot)

		// Next slot at 30 min intervals
		current = current.Add(30 * time.Minute)
	}

	return slots
}

// filterBusySlots removes slots that overlap with busy times
func (sf *SlotFinder) filterBusySlots(slots []entity.TimeSlot, busyTimes []entity.TimeSlot) []entity.TimeSlot {
	filtered := []entity.TimeSlot{}

	for _, slot := range slots {
		isFree := true
		for _, busy := range busyTimes {
			if sf.overlaps(slot, busy) {
				isFree = false
				break
			}
		}
		if isFree {
			filtered = append(filtered, slot)
		}
	}

	return filtered
}

// overlaps checks if two time slots overlap
func (sf *SlotFinder) overlaps(a, b entity.TimeSlot) bool {
	return a.Start.Before(b.End) && a.End.After(b.Start)
}

// scoreSlots assigns scores to slots based on preferences
func (sf *SlotFinder) scoreSlots(
	slots []entity.TimeSlot,
	preferences *entity.EventPreferences,
	totalParticipants int,
) []entity.EventSlot {

	result := make([]entity.EventSlot, len(slots))

	for i, slot := range slots {
		score := 50 // Base score

		hour := slot.Start.Hour()

		// Score based on time preferences
		if preferences != nil {
			if preferences.PreferMorning && hour >= 8 && hour < 12 {
				score += 30
			}
			if preferences.PreferAfternoon && hour >= 13 && hour < 18 {
				score += 30
			}
		}

		// Bonus for 9:00, 10:00, 14:00, 15:00 (common meeting times)
		if hour == 9 || hour == 10 || hour == 14 || hour == 15 {
			score += 20
		}

		// Bonus for weekdays (Mon-Wed higher than Thu-Fri)
		weekday := slot.Start.Weekday()
		if weekday >= time.Monday && weekday <= time.Wednesday {
			score += 15
		} else if weekday == time.Thursday || weekday == time.Friday {
			score += 10
		}

		// Earlier dates get slightly higher scores
		daysFromNow := int(time.Until(slot.Start).Hours() / 24)
		if daysFromNow <= 3 {
			score += 10
		} else if daysFromNow <= 7 {
			score += 5
		}

		result[i] = entity.EventSlot{
			StartTime:         slot.Start,
			EndTime:           slot.End,
			Score:             score,
			AvailableCount:    totalParticipants,
			TotalParticipants: totalParticipants,
		}
	}

	return result
}

// roundToNextHalfHour rounds time to next 00 or 30 minutes
func (sf *SlotFinder) roundToNextHalfHour(t time.Time) time.Time {
	minute := t.Minute()
	if minute == 0 || minute == 30 {
		return t
	}

	if minute < 30 {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 30, 0, 0, t.Location())
	}
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
}
