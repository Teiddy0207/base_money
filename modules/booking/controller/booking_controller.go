package controller

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"go-api-starter/core/config"
	"go-api-starter/core/constants"
	"go-api-starter/core/errors"
	"go-api-starter/core/logger"
	"go-api-starter/core/utils"
	authservice "go-api-starter/modules/auth/service"
	bookingsvc "go-api-starter/modules/booking/service"
	caldto "go-api-starter/modules/calendar/dto"
	calsvc "go-api-starter/modules/calendar/service"
	meetentity "go-api-starter/modules/meeting/entity"
	meetrepo "go-api-starter/modules/meeting/repository"
	notifdto "go-api-starter/modules/notification/dto"
	notifsvc "go-api-starter/modules/notification/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type BookingController struct {
	CalendarService calsvc.CalendarService
	AuthService     authservice.AuthServiceInterface
	MeetingRepo     meetrepo.MeetingRepositoryInterface
	NotificationSvc *notifsvc.NotificationService
	BookingService  bookingsvc.BookingService
}

func NewBookingController(cal calsvc.CalendarService, auth authservice.AuthServiceInterface, meetingRepo meetrepo.MeetingRepositoryInterface, notif *notifsvc.NotificationService, bookingSvc bookingsvc.BookingService) *BookingController {
	return &BookingController{
		CalendarService: cal,
		AuthService:     auth,
		MeetingRepo:     meetingRepo,
		NotificationSvc: notif,
		BookingService:  bookingSvc,
	}
}

func (b *BookingController) PublicPage(c echo.Context) error {
	slug := c.Param("slug")
	html := `
<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Personal Booking</title>
<style>
:root{--bg:#f7f7f9;--fg:#111;--muted:#666;--primary:#2563eb;--border:#ddd}
body{font-family:Inter,Arial,Helvetica,sans-serif;margin:0;background:var(--bg);color:var(--fg)}
.container{max-width:980px;margin:40px auto;padding:0 20px}
.header{display:flex;gap:12px;align-items:center;margin-bottom:24px}
.avatar{width:48px;height:48px;border-radius:999px;background:#e5e7eb;display:flex;align-items:center;justify-content:center;font-weight:700}
.title{font-size:20px;font-weight:700}
.subtitle{color:var(--muted)}
.grid{display:grid;grid-template-columns:1fr 340px;gap:20px}
.card{background:#fff;border:1px solid var(--border);border-radius:12px;padding:16px}
.calendar{display:grid;grid-template-columns:repeat(7,1fr);gap:10px;margin-top:12px}
.day{background:#fff;border:1px solid var(--border);border-radius:10px;height:64px;display:flex;align-items:center;justify-content:center;cursor:pointer}
.day.active{border-color:var(--primary);box-shadow:0 0 0 2px rgba(37,99,235,.2)}
.nav{display:flex;justify-content:space-between;align-items:center;margin-top:8px}
.slots{display:flex;flex-wrap:wrap;gap:8px;margin-top:12px}
.slot{border:1px solid var(--border);border-radius:8px;padding:8px 12px;background:#fff;cursor:pointer}
.slot.active{border-color:var(--primary);background:#eef2ff}
.btn{background:var(--primary);color:#fff;border:none;border-radius:8px;padding:10px 14px;cursor:pointer}
.btn:disabled{opacity:.6;cursor:not-allowed}
.muted{color:var(--muted)}
.row{display:flex;gap:10px;align-items:center;margin-top:10px}
input{padding:8px;border:1px solid var(--border);border-radius:8px;width:100%}
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <div class="avatar">G</div>
    <div>
      <div class="title">` + slug + `</div>
      <div class="subtitle">Schedule a meeting with me</div>
    </div>
  </div>
  <div class="grid">
    <div class="card">
      <div class="nav">
        <button id="prev" class="btn" style="background:#eee;color:#111">‹</button>
        <div id="monthTitle" class="title" style="font-size:18px">Month</div>
        <button id="next" class="btn" style="background:#eee;color:#111">›</button>
      </div>
      <div class="calendar" id="calendar"></div>
      <div class="slots" id="slots"></div>
    </div>
    <div class="card">
      <div class="title" style="font-size:18px">30 min meeting</div>
      <div class="muted">Date TBD<br>Google Meet<br>You'll receive a calendar invitation and meeting link via email</div>
      <div class="row"><input id="name" placeholder="Your name"></div>
      <div class="row"><input id="email" placeholder="Your email"></div>
      <div class="row"><button id="book" class="btn" disabled>Book selected</button></div>
    </div>
  </div>
</div>
<script>
const slug = ` + `"` + slug + `"` + `;
const $ = id => document.getElementById(id)
let current = new Date()
let selectedDay = null
let selectedSlot = null
function monthName(y,m){return new Date(y,m,1).toLocaleString('en-US',{month:'long'})+' '+y}
function startOfMonth(d){return new Date(d.getFullYear(), d.getMonth(), 1)}
function endOfMonth(d){return new Date(d.getFullYear(), d.getMonth()+1, 0)}
function dayStartVN(date){const y=date.getFullYear(),m=('0'+(date.getMonth()+1)).slice(-2),d=('0'+date.getDate()).slice(-2);return y+'-'+m+'-'+d+'T00:00:00+07:00'}
function dayEndVN(date){const y=date.getFullYear(),m=('0'+(date.getMonth()+1)).slice(-2),d=('0'+date.getDate()).slice(-2);return y+'-'+m+'-'+d+'T23:59:59+07:00'}
async function loadSlotsForDay(date){
  selectedSlot=null
  const root=$('slots'); root.innerHTML=''
  const interval=30
  const url='/api/v1/public/booking/'+encodeURIComponent(slug)+'/free?start_time='+encodeURIComponent(dayStartVN(date))+'&end_time='+encodeURIComponent(dayEndVN(date))+'&interval='+encodeURIComponent(interval)
  const res=await fetch(url)
  const data=await res.json()
  const slots=(data&&data.slots)||[]
  slots.forEach(s=>{
    const t=s.start.split('T')[1].slice(0,5)
    const el=document.createElement('div'); el.className='slot'; el.textContent=t
    el.onclick=()=>{selectedSlot={start:s.start, end:s.end}; setActiveSlot(el)}
    root.appendChild(el)
  })
  $('book').disabled=!selectedSlot
}
function setActiveSlot(el){document.querySelectorAll('.slot').forEach(s=>s.classList.remove('active')); el.classList.add('active'); $('book').disabled=false}
function buildCalendar(d){
  const y=d.getFullYear(), m=d.getMonth()
  $('monthTitle').textContent=monthName(y,m)
  const grid=$('calendar'); grid.innerHTML=''
  const first=startOfMonth(d), last=endOfMonth(d)
  const firstDayOfWeek=first.getDay()
  for(let i=0;i<firstDayOfWeek;i++){const ph=document.createElement('div'); ph.className='day'; grid.appendChild(ph)}
  for(let day=1; day<=last.getDate(); day++){
    const date=new Date(y,m,day)
    const el=document.createElement('div'); el.className='day'; el.textContent=day
    el.onclick=()=>{document.querySelectorAll('.day').forEach(d=>d.classList.remove('active')); el.classList.add('active'); selectedDay=date; loadSlotsForDay(date)}
    grid.appendChild(el)
  }
}
$('prev').onclick=()=>{current=new Date(current.getFullYear(), current.getMonth()-1, 1); buildCalendar(current)}
$('next').onclick=()=>{current=new Date(current.getFullYear(), current.getMonth()+1, 1); buildCalendar(current)}
buildCalendar(current)
$('book').onclick=async ()=>{
  if(!selectedSlot) return
  const payload={start_time:selectedSlot.start,end_time:selectedSlot.end,name:$('name').value,email:$('email').value}
  const res=await fetch('/api/v1/public/booking/'+encodeURIComponent(slug)+'/schedule',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)})
  const j=await res.json(); alert((j&&j.message)||'Booked')
}
</script>
</body>
</html>`
	return c.HTML(http.StatusOK, html)
}

func (b *BookingController) PublicPersonalPage(c echo.Context) error {
	id := c.Param("id")
	token := c.QueryParam("token")
	accept := c.QueryParam("accept")
	
	// If token and accept are provided, handle accept action
	if token != "" && accept == "true" {
		return b.handleAcceptFromPersonalPage(c, id, token)
	}
	
	html := `
<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Personal Booking</title>
<style>
:root{--bg:#f7f7f9;--fg:#111;--muted:#666;--primary:#2563eb;--border:#ddd}
body{font-family:Inter,Arial,Helvetica,sans-serif;margin:0;background:var(--bg);color:var(--fg)}
.container{max-width:980px;margin:40px auto;padding:0 20px}
.header{display:flex;gap:12px;align-items:center;margin-bottom:24px}
.avatar{width:48px;height:48px;border-radius:999px;background:#e5e7eb;display:flex;align-items:center;justify-content:center;font-weight:700}
.title{font-size:20px;font-weight:700}
.subtitle{color:var(--muted)}
.grid{display:grid;grid-template-columns:1fr 340px;gap:20px}
.card{background:#fff;border:1px solid var(--border);border-radius:12px;padding:16px}
.calendar{display:grid;grid-template-columns:repeat(7,1fr);gap:10px;margin-top:12px}
.day{background:#fff;border:1px solid var(--border);border-radius:10px;height:64px;display:flex;align-items:center;justify-content:center;cursor:pointer}
.day.active{border-color:var(--primary);box-shadow:0 0 0 2px rgba(37,99,235,.2)}
.nav{display:flex;justify-content:space-between;align-items:center;margin-top:8px}
.slots{display:flex;flex-wrap:wrap;gap:8px;margin-top:12px}
.slot{border:1px solid var(--border);border-radius:8px;padding:8px 12px;background:#fff;cursor:pointer}
.slot.active{border-color:var(--primary);background:#eef2ff}
.btn{background:var(--primary);color:#fff;border:none;border-radius:8px;padding:10px 14px;cursor:pointer}
.btn:disabled{opacity:.6;cursor:not-allowed}
.muted{color:var(--muted)}
.row{display:flex;gap:10px;align-items:center;margin-top:10px}
input{padding:8px;border:1px solid var(--border);border-radius:8px;width:100%}
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <div class="avatar">G</div>
    <div>
      <div class="title">Personal Booking</div>
      <div class="subtitle">Schedule a meeting with me</div>
    </div>
  </div>
  <div class="grid">
    <div class="card">
      <div class="nav">
        <button id="prev" class="btn" style="background:#eee;color:#111">‹</button>
        <div id="monthTitle" class="title" style="font-size:18px">Month</div>
        <button id="next" class="btn" style="background:#eee;color:#111">›</button>
      </div>
      <div class="calendar" id="calendar"></div>
      <div class="slots" id="slots"></div>
    </div>
    <div class="card">
      <div class="title" style="font-size:18px">30 min meeting</div>
      <div class="muted">Date TBD<br>Google Meet<br>You'll receive a calendar invitation and meeting link via email</div>
      <div class="row"><input id="name" placeholder="Your name"></div>
      <div class="row"><input id="email" placeholder="Your email"></div>
      <div class="row"><button id="book" class="btn" disabled>Book selected</button></div>
    </div>
  </div>
</div>
<script>
const id = ` + `"` + id + `"` + `;
const $ = id => document.getElementById(id)
let current = new Date()
let selectedDay = null
let selectedSlot = null
function monthName(y,m){return new Date(y,m,1).toLocaleString('en-US',{month:'long'})+' '+y}
function startOfMonth(d){return new Date(d.getFullYear(), d.getMonth(), 1)}
function endOfMonth(d){return new Date(d.getFullYear(), d.getMonth()+1, 0)}
function dayStartVN(date){const y=date.getFullYear(),m=('0'+(date.getMonth()+1)).slice(-2),d=('0'+date.getDate()).slice(-2);return y+'-'+m+'-'+d+'T00:00:00+07:00'}
function dayEndVN(date){const y=date.getFullYear(),m=('0'+(date.getMonth()+1)).slice(-2),d=('0'+date.getDate()).slice(-2);return y+'-'+m+'-'+d+'T23:59:59+07:00'}
async function loadSlotsForDay(date){
  selectedSlot=null
  const root=$('slots'); root.innerHTML=''
  try {
  const body={duration_minutes:30, days_ahead:1, start_date: date.toISOString().split('T')[0], time_preference:'', working_hours_only:false}
  const res=await fetch('/api/v1/public/booking/'+encodeURIComponent(id)+'/suggested-slots',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)})
  const data=await res.json()
  const slots=(data&&data.slots)||[]
  // Helper function to convert UTC time to VN timezone and format
  const formatTimeVN = (timeStr) => {
    const d = new Date(timeStr)
    // Use Intl.DateTimeFormat to get VN time components accurately
    const formatter = new Intl.DateTimeFormat('en-CA', {
      timeZone: 'Asia/Ho_Chi_Minh',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false
    })
    const parts = formatter.formatToParts(d)
    const y = parts.find(p => p.type === 'year').value
    const m = parts.find(p => p.type === 'month').value
    const day = parts.find(p => p.type === 'day').value
    const h = parts.find(p => p.type === 'hour').value
    const min = parts.find(p => p.type === 'minute').value
    const sec = parts.find(p => p.type === 'second').value
    return {date: y + '-' + m + '-' + day, time: h + ':' + min, full: y + '-' + m + '-' + day + 'T' + h + ':' + min + ':' + sec + '+07:00'}
  }
  
  slots.forEach(s=>{
    const startTime = s.start_time || ''
    const endTime = s.end_time || ''
    // Convert to VN timezone for display
    const startVN = formatTimeVN(startTime)
    // Display time (HH:mm)
    const el=document.createElement('div'); el.className='slot'; el.textContent=startVN.time
    el.onclick=()=>{
      // When user selects slot, use the same VN timezone format
      const endVN = formatTimeVN(endTime)
      selectedSlot={start:startVN.full, end:endVN.full}
      setActiveSlot(el)
    }
    root.appendChild(el)
  })
  } catch (e) {
    const el=document.createElement('div'); el.className='muted'; el.textContent='Failed to load slots'
    root.appendChild(el)
  }
  $('book').disabled=!selectedSlot
}
function setActiveSlot(el){document.querySelectorAll('.slot').forEach(s=>s.classList.remove('active')); el.classList.add('active'); $('book').disabled=false}
function buildCalendar(d){
  const y=d.getFullYear(), m=d.getMonth()
  $('monthTitle').textContent=monthName(y,m)
  const grid=$('calendar'); grid.innerHTML=''
  const first=startOfMonth(d), last=endOfMonth(d)
  const firstDayOfWeek=first.getDay()
  for(let i=0;i<firstDayOfWeek;i++){const ph=document.createElement('div'); ph.className='day'; grid.appendChild(ph)}
  for(let day=1; day<=last.getDate(); day++){
    const date=new Date(y,m,day)
    const el=document.createElement('div'); el.className='day'; el.textContent=day
    el.onclick=()=>{document.querySelectorAll('.day').forEach(d=>d.classList.remove('active')); el.classList.add('active'); selectedDay=date; loadSlotsForDay(date)}
    grid.appendChild(el)
  }
}
$('prev').onclick=()=>{current=new Date(current.getFullYear(), current.getMonth()-1, 1); buildCalendar(current)}
$('next').onclick=()=>{current=new Date(current.getFullYear(), current.getMonth()+1, 1); buildCalendar(current)}
buildCalendar(current)
$('book').onclick=async ()=>{
  if(!selectedSlot) return
  const payload={start_time:selectedSlot.start,end_time:selectedSlot.end,name:$('name').value,email:$('email').value}
  const res=await fetch('/api/v1/public/booking/'+encodeURIComponent(id)+'/schedule',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)})
  const j=await res.json(); alert((j&&j.message)||'Booked')
}
</script>
</body>
</html>`
	return c.HTML(http.StatusOK, html)
}

// handleAcceptFromPersonalPage handles accept action from personal booking page
func (b *BookingController) handleAcceptFromPersonalPage(c echo.Context, idStr, token string) error {
	ctx := c.Request().Context()
	eventID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid event id", nil))
	}
	
	ev, err := b.MeetingRepo.GetEventByID(ctx, eventID)
	if err != nil || ev == nil {
		return c.JSON(http.StatusNotFound, errors.NewAppError(errors.ErrNotFound, "event not found", err))
	}
	
	claims, err := utils.ValidateAndParseToken(token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "invalid token", err))
	}
	
	if ev.HostID == nil || *ev.HostID != claims.UserID {
		return c.JSON(http.StatusForbidden, errors.NewAppError(errors.ErrForbidden, "not authorized", nil))
	}
	
	if ev.StartDate == nil || ev.EndDate == nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "missing start/end", nil))
	}
	
	guestEmail := ""
	if ev.Preferences != nil && *ev.Preferences != "" {
		type Pref struct {
			GuestName  string `json:"guest_name"`
			GuestEmail string `json:"guest_email"`
		}
		var p Pref
		_ = json.Unmarshal([]byte(*ev.Preferences), &p)
		guestEmail = strings.TrimSpace(p.GuestEmail)
	}
	
	// Get timezone, default to Asia/Ho_Chi_Minh if empty
	timezone := ev.Timezone
	if timezone == "" {
		timezone = "Asia/Ho_Chi_Minh"
	}
	
	// IMPORTANT: Add 1 day to the booking time when accepting
	// User selects time, but when accepting, automatically add 1 day
	adjustedStartDate := ev.StartDate.AddDate(0, 0, 1)
	adjustedEndDate := ev.EndDate.AddDate(0, 0, 1)
	
	logger.Info("handleAcceptFromPersonalPage:TimeAdjusted",
		"event_id", eventID.String(),
		"original_start", ev.StartDate.Format(time.RFC3339),
		"adjusted_start", adjustedStartDate.Format(time.RFC3339),
		"original_end", ev.EndDate.Format(time.RFC3339),
		"adjusted_end", adjustedEndDate.Format(time.RFC3339))
	
	req := &caldto.CreateEventRequest{
		Title:       ev.Title,
		Description: "Personal booking",
		StartTime:   formatTimeInTimezone(adjustedStartDate, timezone),
		EndTime:     formatTimeInTimezone(adjustedEndDate, timezone),
		Timezone:    timezone,
	}
	if guestEmail != "" {
		req.Attendees = []string{guestEmail}
	}
	
	created, er := b.CalendarService.CreateEvent(ctx, claims.UserID, req)
	if er != nil {
		return c.JSON(http.StatusForbidden, errors.NewAppError(errors.ErrForbidden, er.Error(), er))
	}
	
	ev.Status = meetentity.EventStatusScheduled
	if created.MeetingLink != "" {
		link := created.MeetingLink
		ev.MeetingLink = &link
	}
	if err := b.MeetingRepo.UpdateEvent(ctx, ev); err != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, "failed to update event", err))
	}

	// Format event time for display (with +1 day adjustment)
	eventTimeStr := "Chưa xác định"
	if ev.StartDate != nil && ev.EndDate != nil {
		// Add 1 day for display (same as when creating calendar event)
		adjustedStartDate := ev.StartDate.AddDate(0, 0, 1)
		adjustedEndDate := ev.EndDate.AddDate(0, 0, 1)
		// Convert to VN timezone for display
		vnLoc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
		startVN := adjustedStartDate.In(vnLoc)
		endVN := adjustedEndDate.In(vnLoc)
		startTime := startVN.Format("15:04")
		endTime := endVN.Format("15:04")
		dateStr := startVN.Format("02/01/2006")
		eventTimeStr = fmt.Sprintf("%s, %s - %s", dateStr, startTime, endTime)
	}

	// Return HTML success page
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="vi">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Lịch đã được chấp nhận | SmartMeet</title>
	<style>
		* {
			margin: 0;
			padding: 0;
			box-sizing: border-box;
		}
		
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
			min-height: 100vh;
			display: flex;
			align-items: center;
			justify-content: center;
			padding: 20px;
		}
		
		.container {
			background: white;
			border-radius: 20px;
			box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
			padding: 48px;
			max-width: 500px;
			width: 100%%;
			text-align: center;
		}
		
		.success-icon {
			width: 80px;
			height: 80px;
			border-radius: 50%%;
			background: #10b981;
			display: flex;
			align-items: center;
			justify-content: center;
			margin: 0 auto 24px;
			animation: scaleIn 0.5s ease-out;
		}
		
		.success-icon::before {
			content: "✓";
			color: white;
			font-size: 48px;
			font-weight: bold;
		}
		
		@keyframes scaleIn {
			from {
				transform: scale(0);
			}
			to {
				transform: scale(1);
			}
		}
		
		h1 {
			font-size: 28px;
			color: #1e293b;
			margin-bottom: 12px;
			font-weight: 700;
		}
		
		.message {
			font-size: 16px;
			color: #64748b;
			margin-bottom: 32px;
			line-height: 1.6;
		}
		
		.event-details {
			background: #f8fafc;
			border-radius: 12px;
			padding: 24px;
			margin-bottom: 32px;
			text-align: left;
		}
		
		.event-details h2 {
			font-size: 18px;
			color: #1e293b;
			margin-bottom: 16px;
			font-weight: 600;
		}
		
		.detail-row {
			display: flex;
			align-items: center;
			margin-bottom: 12px;
			font-size: 14px;
			color: #475569;
		}
		
		.detail-row:last-child {
			margin-bottom: 0;
		}
		
		.detail-label {
			font-weight: 600;
			color: #64748b;
			min-width: 100px;
		}
		
		.detail-value {
			color: #1e293b;
			flex: 1;
		}
		
		.meeting-link {
			display: inline-block;
			background: #2563eb;
			color: white;
			text-decoration: none;
			padding: 12px 24px;
			border-radius: 8px;
			font-weight: 600;
			margin-top: 8px;
			transition: background 0.2s;
		}
		
		.meeting-link:hover {
			background: #1d4ed8;
		}
		
		.close-btn {
			background: #e2e8f0;
			color: #475569;
			border: none;
			padding: 12px 24px;
			border-radius: 8px;
			font-weight: 600;
			cursor: pointer;
			transition: background 0.2s;
		}
		
		.close-btn:hover {
			background: #cbd5e1;
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="success-icon"></div>
		<h1>Lịch đã được chấp nhận!</h1>
		<p class="message">Yêu cầu đặt lịch của bạn đã được chấp nhận. Sự kiện đã được thêm vào lịch.</p>
		
		<div class="event-details">
			<h2>Chi tiết sự kiện</h2>
			<div class="detail-row">
				<span class="detail-label">Tiêu đề:</span>
				<span class="detail-value">%s</span>
			</div>
			<div class="detail-row">
				<span class="detail-label">Thời gian:</span>
				<span class="detail-value">%s</span>
			</div>
			%s
		</div>
		
		<button class="close-btn" onclick="window.close()">Đóng</button>
	</div>
</body>
</html>`,
		html.EscapeString(ev.Title),
		eventTimeStr,
		func() string {
			if ev.MeetingLink != nil && *ev.MeetingLink != "" {
				return fmt.Sprintf(`
			<div class="detail-row">
				<span class="detail-label">Meeting Link:</span>
				<span class="detail-value">
					<a href="%s" target="_blank" class="meeting-link">Tham gia Google Meet</a>
				</span>
			</div>`, html.EscapeString(*ev.MeetingLink))
			}
			return ""
		}(),
	)

	return c.HTML(http.StatusOK, html)
}

func (b *BookingController) PublicSuggestedSlots(c echo.Context) error {
	ctx := c.Request().Context()
	idStr := c.Param("id")
	slID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid social login id", nil))
	}
	userID, appErr := b.AuthService.GetUserIDBySocialLoginID(ctx, slID)
	if appErr != nil {
		return c.JSON(http.StatusNotFound, appErr)
	}
	var req caldto.SuggestedSlotsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid body", nil))
	}
	req.UserIDs = []string{userID.String()}
	res, err2 := b.CalendarService.FindAvailableSlots(ctx, &req)
	if err2 != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, err2.Error(), err2))
	}
	return c.JSON(http.StatusOK, res)
}

func (b *BookingController) PublicFreeSlots(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("slug")
	startStr := c.QueryParam("start_time")
	endStr := c.QueryParam("end_time")
	intervalStr := c.QueryParam("interval")
	window := c.QueryParam("window")
	if startStr == "" || endStr == "" {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "start_time and end_time are required", nil))
	}
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid start_time", nil))
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid end_time", nil))
	}
	interval := 30
	if intervalStr != "" {
		interval = utils.ToNumberWithDefault(intervalStr, 30)
	}
	slID, ok := tryParseUUID(slug)
	var userID uuid.UUID
	if ok {
		uid, appErr := b.AuthService.GetUserIDBySocialLoginID(ctx, slID)
		if appErr != nil {
			return c.JSON(http.StatusNotFound, appErr)
		}
		userID = uid
	} else {
		sl, appErr := b.AuthService.GetSocialLoginBySlug(ctx, slug)
		if appErr != nil || sl == nil {
			return c.JSON(http.StatusNotFound, errors.NewAppError(errors.ErrNotFound, "not found", nil))
		}
		userID = sl.UserID
	}
	busy, ferr := b.CalendarService.GetFreeBusy(ctx, userID, start, end)
	if ferr != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, ferr.Error(), ferr))
	}
	slots := computeFreeSlots(start, end, busy, interval, window)
	return c.JSON(http.StatusOK, map[string]any{"slots": slots})
}

func (b *BookingController) PublicSchedule(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("slug")
	var req struct {
		Name      string `json:"name"`
		Email     string `json:"email"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid body", nil))
	}
	// Parse time from RFC3339 format (e.g., "2026-01-28T12:00:00+07:00")
	// This preserves the timezone information
	logger.Info("PublicSchedule:ParseTime",
		"start_time_string", req.StartTime,
		"end_time_string", req.EndTime)
	
	start, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid start_time", nil))
	}
	end, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid end_time", nil))
	}
	
	// Log parsed time values
	logger.Info("PublicSchedule:ParsedTime",
		"start_utc", start.UTC().Format(time.RFC3339),
		"start_local", start.Format(time.RFC3339),
		"end_utc", end.UTC().Format(time.RFC3339),
		"end_local", end.Format(time.RFC3339))
	
	// IMPORTANT: When time is parsed from RFC3339 with timezone, Go creates a time.Time
	// with the correct absolute value (Unix timestamp). However, when saving to PostgreSQL
	// (TIMESTAMP WITH TIME ZONE), we need to ensure the time is in UTC to avoid confusion.
	// Convert to UTC explicitly to ensure correct storage.
	start = start.UTC()
	end = end.UTC()
	
	logger.Info("PublicSchedule:TimeConvertedToUTC",
		"start_utc", start.Format(time.RFC3339),
		"end_utc", end.Format(time.RFC3339))
	
	slID, ok := tryParseUUID(slug)
	var userID uuid.UUID
	var hostEmail string
	if ok {
		uid, appErr := b.AuthService.GetUserIDBySocialLoginID(ctx, slID)
		if appErr != nil {
			return c.JSON(http.StatusNotFound, appErr)
		}
		userID = uid
		if sl, appErr := b.AuthService.GetSocialLoginByID(ctx, slID); appErr == nil && sl != nil && sl.ProviderEmail != nil {
			hostEmail = strings.TrimSpace(*sl.ProviderEmail)
		}
	} else {
		sl, appErr := b.AuthService.GetSocialLoginBySlug(ctx, slug)
		if appErr != nil || sl == nil {
			return c.JSON(http.StatusNotFound, errors.NewAppError(errors.ErrNotFound, "not found", nil))
		}
		userID = sl.UserID
		if sl.ProviderEmail != nil {
			hostEmail = strings.TrimSpace(*sl.ProviderEmail)
		}
	}
	// Create pending event record
	title := "Booking with " + strings.TrimSpace(req.Name)
	ev := &meetentity.Event{
		HostID:          &userID,
		Title:           title,
		DurationMinutes: 30,
		Status:          meetentity.EventStatusPending,
		Timezone:        "Asia/Ho_Chi_Minh",
	}
	created, errCreate := b.MeetingRepo.CreateEvent(ctx, ev)
	if errCreate != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, "failed to create booking request", errCreate))
	}
	// Update time window on event, keep status pending
	created.StartDate = &start
	created.EndDate = &end
	if errUpd := b.MeetingRepo.UpdateEvent(ctx, created); errUpd != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, "failed to set booking time", errUpd))
	}
	// Notify host
	if b.NotificationSvc != nil {
		_ = b.NotificationSvc.Create(ctx, &notifdto.CreateNotificationRequest{
			UserID:  userID,
			Title:   "Yêu cầu đặt lịch mới",
			Message: title,
			Type:    "booking_request",
			Data: map[string]interface{}{
				"event_id":    created.ID.String(),
				"start_time":  req.StartTime,
				"end_time":    req.EndTime,
				"guest_name":  req.Name,
				"guest_email": strings.TrimSpace(req.Email),
			},
		})
	}
	// Send email to host if available
	if utils.IsValidEmail(hostEmail) {
		conf := utils.GetEmailConfig()
		approveToken, _ := utils.GenerateToken(userID, &hostEmail, nil, "booking_approval", 15*time.Minute)
		declineToken, _ := utils.GenerateToken(userID, &hostEmail, nil, "booking_approval", 15*time.Minute)
		base := config.Get().Server.BaseURL
		if base == "" {
			base = "http://" + config.Get().Server.Host + ":" + fmt.Sprint(config.Get().Server.Port)
		}
		acceptURL := base + "/api/v1/public/booking/requests/" + created.ID.String() + "/accept?token=" + approveToken
		declineURL := base + "/api/v1/public/booking/requests/" + created.ID.String() + "/decline?token=" + declineToken
		// Format time in VN timezone for email (with +1 day adjustment - same as when accepting)
		// Add 1 day to show the actual time that will be scheduled when accepted
		adjustedStart := start.AddDate(0, 0, 1)
		adjustedEnd := end.AddDate(0, 0, 1)
		vnLoc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
		startVN := adjustedStart.In(vnLoc)
		endVN := adjustedEnd.In(vnLoc)
		timeStr := fmt.Sprintf("%s %s - %s", startVN.Format("02/01/2006"), startVN.Format("15:04"), endVN.Format("15:04"))
		body := "<h3>New booking request</h3><p>Guest: " + templateEscape(req.Name) + " (" + templateEscape(strings.TrimSpace(req.Email)) + ")</p><p>Time: " + templateEscape(timeStr) + "</p><p><a href=\"" + templateEscape(acceptURL) + "\">Accept</a> &nbsp;|&nbsp; <a href=\"" + templateEscape(declineURL) + "\">Decline</a></p>"
		_ = utils.SendEmailTLS(*conf, utils.EmailMessage{
			To:      []string{hostEmail},
			Subject: "New booking request",
			Body:    body,
			IsHTML:  true,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"message":  "Booking request sent",
		"event_id": created.ID.String(),
		"status":   "pending",
	})
}

func tryParseUUID(s string) (uuid.UUID, bool) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// getUserIDFromContext extracts user ID from JWT context
func (b *BookingController) getUserIDFromContext(c echo.Context) (uuid.UUID, error) {
	tokenData := c.Get(constants.ContextTokenData)
	if tokenData == nil {
		return uuid.Nil, errors.NewAppError(errors.ErrUnauthorized, "User not authenticated", nil)
	}

	claims, ok := tokenData.(*utils.TokenClaims)
	if !ok {
		return uuid.Nil, errors.NewAppError(errors.ErrUnauthorized, "Invalid token data", nil)
	}

	return claims.UserID, nil
}

// GetPersonalBookingURL returns the personal booking page URL for the authenticated user
// GET /api/v1/private/booking/personal-url
func (b *BookingController) GetPersonalBookingURL(c echo.Context) error {
	ctx := c.Request().Context()

	// Get current user ID from context
	userID, err := b.getUserIDFromContext(c)
	if err != nil {
		logger.Error("BookingController:GetPersonalBookingURL:GetUserIDFromContext:Error", "error", err)
		return c.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "User not authenticated", nil))
	}

	logger.Info("BookingController:GetPersonalBookingURL:Start", "user_id", userID)

	// Call service to get personal booking URL
	result, appErr := b.BookingService.GetPersonalBookingURL(ctx, userID)
	if appErr != nil {
		logger.Error("BookingController:GetPersonalBookingURL:Service:Error", "error", appErr, "user_id", userID)

		// Map error codes to HTTP status
		httpStatus := http.StatusInternalServerError
		if appErr.Code == errors.ErrNotFound {
			httpStatus = http.StatusNotFound
		} else if appErr.Code == errors.ErrUnauthorized {
			httpStatus = http.StatusUnauthorized
		}

		return c.JSON(httpStatus, errors.NewAppError(appErr.Code, appErr.Message, appErr.Err))
	}

	logger.Info("BookingController:GetPersonalBookingURL:Success", "user_id", userID, "url", result.URL)

	// Return JSON response with standard format
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    http.StatusOK,
		"message":   "Lấy URL đặt lịch thành công",
		"data":      result,
		"timestamp": time.Now(),
	})
}

// GetWeekStatistics returns weekly event statistics
// GET /api/v1/private/booking/week-statistics
func (b *BookingController) GetWeekStatistics(c echo.Context) error {
	ctx := c.Request().Context()

	// Get current user ID from context
	userID, err := b.getUserIDFromContext(c)
	if err != nil {
		logger.Error("BookingController:GetWeekStatistics:GetUserIDFromContext:Error", "error", err)
		return c.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "User not authenticated", nil))
	}

	logger.Info("BookingController:GetWeekStatistics:Start", "user_id", userID)

	// Call service to get week statistics
	result, appErr := b.BookingService.GetWeekStatistics(ctx, userID)
	if appErr != nil {
		logger.Error("BookingController:GetWeekStatistics:Service:Error", "error", appErr, "user_id", userID)

		// Map error codes to HTTP status
		httpStatus := http.StatusInternalServerError
		if appErr.Code == errors.ErrNotFound {
			httpStatus = http.StatusNotFound
		} else if appErr.Code == errors.ErrUnauthorized {
			httpStatus = http.StatusUnauthorized
		}

		return c.JSON(httpStatus, errors.NewAppError(appErr.Code, appErr.Message, appErr.Err))
	}

	logger.Info("BookingController:GetWeekStatistics:Success", "user_id", userID, "total_events", result.TotalEvents, "total_duration_minutes", result.TotalDurationMinutes)

	// Return JSON response with standard format
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    http.StatusOK,
		"message":   "Lấy thống kê tuần thành công",
		"data":      result,
		"timestamp": time.Now(),
	})
}

func templateEscape(s string) string {
	return html.EscapeString(s)
}



func formatTimeInTimezone(t time.Time, timezone string) string {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Fallback to Asia/Ho_Chi_Minh if timezone is invalid
		loc, _ = time.LoadLocation("Asia/Ho_Chi_Minh")
	}
	
	// Convert UTC time to the target timezone to get the local representation
	// This gives us the correct date/time components in that timezone
	tInTZ := t.In(loc)
	
	// Format as RFC3339 WITH timezone offset (+07:00)
	// This ensures Google Calendar receives the time with correct timezone information
	return tInTZ.Format(time.RFC3339)
}

func (b *BookingController) PrivateListPending(c echo.Context) error {
	tokenData := c.Get(constants.ContextTokenData)
	if tokenData == nil {
		return c.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "unauthorized", nil))
	}
	claims, ok := tokenData.(*utils.TokenClaims)
	if !ok {
		return c.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "unauthorized", nil))
	}
	events, err := b.MeetingRepo.GetEventsByHostID(c.Request().Context(), claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, "failed to get events", err))
	}
	res := make([]map[string]any, 0, len(events))
	for _, e := range events {
		if e.Status != meetentity.EventStatusPending {
			continue
		}
		res = append(res, map[string]any{
			"id":         e.ID.String(),
			"title":      e.Title,
			"start_time": e.StartDate,
			"end_time":   e.EndDate,
			"status":     e.Status,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"items": res})
}

// PrivateAcceptRequest accepts a booking request
// @Summary Chấp nhận yêu cầu đặt lịch
// @Description Chấp nhận yêu cầu đặt lịch và tạo sự kiện trên Calendar
// @Tags Booking
// @Security BearerAuth
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.AppError
// @Failure 401 {object} errors.AppError
// @Router /private/booking/{id}/accept [post]
func (b *BookingController) PrivateAcceptRequest(c echo.Context) error {
	tokenData := c.Get(constants.ContextTokenData)
	if tokenData == nil {
		return c.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "unauthorized", nil))
	}
	claims, ok := tokenData.(*utils.TokenClaims)
	if !ok {
		return c.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "unauthorized", nil))
	}
	id := c.Param("id")
	eventID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid event id", nil))
	}
	ev, err := b.MeetingRepo.GetEventByID(c.Request().Context(), eventID)
	if err != nil || ev == nil {
		return c.JSON(http.StatusNotFound, errors.NewAppError(errors.ErrNotFound, "event not found", err))
	}
	if ev.HostID == nil || *ev.HostID != claims.UserID {
		return c.JSON(http.StatusForbidden, errors.NewAppError(errors.ErrForbidden, "not authorized", nil))
	}
	if ev.StartDate == nil || ev.EndDate == nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "missing start/end", nil))
	}
	// Extract guest email from preferences if available
	guestEmail := ""
	if ev.Preferences != nil && *ev.Preferences != "" {
		type Pref struct {
			GuestName  string `json:"guest_name"`
			GuestEmail string `json:"guest_email"`
		}
		var p Pref
		_ = json.Unmarshal([]byte(*ev.Preferences), &p)
		guestEmail = strings.TrimSpace(p.GuestEmail)
	}
	
	// Get timezone, default to Asia/Ho_Chi_Minh if empty
	timezone := ev.Timezone
	if timezone == "" {
		timezone = "Asia/Ho_Chi_Minh"
	}
	
	// IMPORTANT: Add 1 day to the booking time when accepting
	// User selects time, but when accepting, automatically add 1 day
	adjustedStartDate := ev.StartDate.AddDate(0, 0, 1)
	adjustedEndDate := ev.EndDate.AddDate(0, 0, 1)
	
	logger.Info("PrivateAcceptRequest:TimeAdjusted",
		"event_id", eventID.String(),
		"original_start", ev.StartDate.Format(time.RFC3339),
		"adjusted_start", adjustedStartDate.Format(time.RFC3339),
		"original_end", ev.EndDate.Format(time.RFC3339),
		"adjusted_end", adjustedEndDate.Format(time.RFC3339))
	
	req := &caldto.CreateEventRequest{
		Title:       ev.Title,
		Description: "Personal booking",
		StartTime:   formatTimeInTimezone(adjustedStartDate, timezone),
		EndTime:     formatTimeInTimezone(adjustedEndDate, timezone),
		Timezone:    timezone,
	}
	if guestEmail != "" {
		req.Attendees = []string{guestEmail}
	}
	created, er := b.CalendarService.CreateEvent(c.Request().Context(), claims.UserID, req)
	if er != nil {
		return c.JSON(http.StatusForbidden, errors.NewAppError(errors.ErrForbidden, er.Error(), er))
	}
	// Update event status and meeting_link
	ev.Status = meetentity.EventStatusScheduled
	if created.MeetingLink != "" {
		link := created.MeetingLink
		ev.MeetingLink = &link
	}
	if err := b.MeetingRepo.UpdateEvent(c.Request().Context(), ev); err != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, "failed to update event", err))
	}
	// Email guest if available
	if utils.IsValidEmail(guestEmail) {
		conf := utils.GetEmailConfig()
		link := ""
		if created.MeetingLink != "" {
			link = created.MeetingLink
		}
		// Format time in VN timezone for email (human-readable format, with +1 day adjustment)
		// Add 1 day for email display (same as when creating calendar event)
		adjustedStartDate := ev.StartDate.AddDate(0, 0, 1)
		adjustedEndDate := ev.EndDate.AddDate(0, 0, 1)
		vnLoc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
		startVN := adjustedStartDate.In(vnLoc)
		endVN := adjustedEndDate.In(vnLoc)
		timeStr := fmt.Sprintf("%s, %s - %s", startVN.Format("02/01/2006"), startVN.Format("15:04"), endVN.Format("15:04"))
		body := "<h3>Booking confirmed</h3><p>Title: " + templateEscape(ev.Title) + "</p><p>Time: " + templateEscape(timeStr) + "</p><p>Meeting link: " + templateEscape(link) + "</p>"
		_ = utils.SendEmailTLS(*conf, utils.EmailMessage{
			To:      []string{guestEmail},
			Subject: "Your meeting is confirmed",
			Body:    body,
			IsHTML:  true,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"message": "accepted", "event_id": ev.ID.String()})
}

// PrivateDeclineRequest declines a booking request
// @Summary Từ chối yêu cầu đặt lịch
// @Description Từ chối yêu cầu đặt lịch
// @Tags Booking
// @Security BearerAuth
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.AppError
// @Failure 401 {object} errors.AppError
// @Router /private/booking/{id}/decline [post]
func (b *BookingController) PrivateDeclineRequest(c echo.Context) error {
	tokenData := c.Get(constants.ContextTokenData)
	if tokenData == nil {
		return c.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "unauthorized", nil))
	}
	claims, ok := tokenData.(*utils.TokenClaims)
	if !ok {
		return c.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "unauthorized", nil))
	}
	id := c.Param("id")
	eventID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid event id", nil))
	}
	ev, err := b.MeetingRepo.GetEventByID(c.Request().Context(), eventID)
	if err != nil || ev == nil {
		return c.JSON(http.StatusNotFound, errors.NewAppError(errors.ErrNotFound, "event not found", err))
	}
	if ev.HostID == nil || *ev.HostID != claims.UserID {
		return c.JSON(http.StatusForbidden, errors.NewAppError(errors.ErrForbidden, "not authorized", nil))
	}
	ev.Status = meetentity.EventStatusCancelled
	if err := b.MeetingRepo.UpdateEvent(c.Request().Context(), ev); err != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, "failed to update event", err))
	}
	// Email guest if available
	guestEmail := ""
	if ev.Preferences != nil && *ev.Preferences != "" {
		type Pref struct {
			GuestName  string `json:"guest_name"`
			GuestEmail string `json:"guest_email"`
		}
		var p Pref
		_ = json.Unmarshal([]byte(*ev.Preferences), &p)
		guestEmail = strings.TrimSpace(p.GuestEmail)
	}
	if utils.IsValidEmail(guestEmail) {
		conf := utils.GetEmailConfig()
		body := "<h3>Booking declined</h3><p>Title: " + templateEscape(ev.Title) + "</p>"
		_ = utils.SendEmailTLS(*conf, utils.EmailMessage{
			To:      []string{guestEmail},
			Subject: "Your meeting request was declined",
			Body:    body,
			IsHTML:  true,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"message": "declined"})
}

func (b *BookingController) PublicTokenAccept(c echo.Context) error {
	id := c.Param("id")
	token := c.QueryParam("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "missing token", nil))
	}
	eventID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid event id", nil))
	}
	ev, err := b.MeetingRepo.GetEventByID(c.Request().Context(), eventID)
	if err != nil || ev == nil {
		return c.JSON(http.StatusNotFound, errors.NewAppError(errors.ErrNotFound, "event not found", err))
	}
	claims, err := utils.ValidateAndParseToken(token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "invalid token", err))
	}
	if ev.HostID == nil || *ev.HostID != claims.UserID {
		return c.JSON(http.StatusForbidden, errors.NewAppError(errors.ErrForbidden, "not authorized", nil))
	}
	if ev.StartDate == nil || ev.EndDate == nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "missing start/end", nil))
	}
	guestEmail := ""
	if ev.Preferences != nil && *ev.Preferences != "" {
		type Pref struct {
			GuestName  string `json:"guest_name"`
			GuestEmail string `json:"guest_email"`
		}
		var p Pref
		_ = json.Unmarshal([]byte(*ev.Preferences), &p)
		guestEmail = strings.TrimSpace(p.GuestEmail)
	}
	
	// Get timezone, default to Asia/Ho_Chi_Minh if empty
	timezone := ev.Timezone
	if timezone == "" {
		timezone = "Asia/Ho_Chi_Minh"
	}
	
	// IMPORTANT: Add 1 day to the booking time when accepting
	// User selects time, but when accepting, automatically add 1 day
	adjustedStartDate := ev.StartDate.AddDate(0, 0, 1)
	adjustedEndDate := ev.EndDate.AddDate(0, 0, 1)
	
	// Log the time values for debugging
	logger.Info("PublicTokenAccept:FormattingTime",
		"event_id", eventID.String(),
		"original_start_utc", ev.StartDate.Format(time.RFC3339),
		"original_end_utc", ev.EndDate.Format(time.RFC3339),
		"adjusted_start_utc", adjustedStartDate.Format(time.RFC3339),
		"adjusted_end_utc", adjustedEndDate.Format(time.RFC3339),
		"timezone", timezone)
	
	startTimeFormatted := formatTimeInTimezone(adjustedStartDate, timezone)
	endTimeFormatted := formatTimeInTimezone(adjustedEndDate, timezone)
	
	logger.Info("PublicTokenAccept:FormattedTime",
		"event_id", eventID.String(),
		"start_time_formatted", startTimeFormatted,
		"end_time_formatted", endTimeFormatted)
	
	req := &caldto.CreateEventRequest{
		Title:       ev.Title,
		Description: "Personal booking",
		StartTime:   startTimeFormatted,
		EndTime:     endTimeFormatted,
		Timezone:    timezone,
	}
	if guestEmail != "" {
		req.Attendees = []string{guestEmail}
	}
	created, er := b.CalendarService.CreateEvent(c.Request().Context(), claims.UserID, req)
	if er != nil {
		return c.JSON(http.StatusForbidden, errors.NewAppError(errors.ErrForbidden, er.Error(), er))
	}
	ev.Status = meetentity.EventStatusScheduled
	if created.MeetingLink != "" {
		link := created.MeetingLink
		ev.MeetingLink = &link
	}
	if err := b.MeetingRepo.UpdateEvent(c.Request().Context(), ev); err != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, "failed to update event", err))
	}

	// Format event time for display (with +1 day adjustment)
	eventTimeStr := "Chưa xác định"
	if ev.StartDate != nil && ev.EndDate != nil {
		// Add 1 day for display (same as when creating calendar event)
		adjustedStartDate := ev.StartDate.AddDate(0, 0, 1)
		adjustedEndDate := ev.EndDate.AddDate(0, 0, 1)
		// Convert to VN timezone for display
		vnLoc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
		startVN := adjustedStartDate.In(vnLoc)
		endVN := adjustedEndDate.In(vnLoc)
		startTime := startVN.Format("15:04")
		endTime := endVN.Format("15:04")
		dateStr := startVN.Format("02/01/2006")
		eventTimeStr = fmt.Sprintf("%s, %s - %s", dateStr, startTime, endTime)
	}

	// Return HTML success page instead of JSON
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="vi">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Lịch đã được chấp nhận | SmartMeet</title>
	<style>
		* {
			margin: 0;
			padding: 0;
			box-sizing: border-box;
		}
		
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
			min-height: 100vh;
			display: flex;
			align-items: center;
			justify-content: center;
			padding: 20px;
		}
		
		.container {
			background: white;
			border-radius: 20px;
			box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
			padding: 48px;
			max-width: 500px;
			width: 100%%;
			text-align: center;
		}
		
		.success-icon {
			width: 80px;
			height: 80px;
			border-radius: 50%%;
			background: #10b981;
			display: flex;
			align-items: center;
			justify-content: center;
			margin: 0 auto 24px;
			animation: scaleIn 0.5s ease-out;
		}
		
		.success-icon::before {
			content: "✓";
			color: white;
			font-size: 48px;
			font-weight: bold;
		}
		
		@keyframes scaleIn {
			from {
				transform: scale(0);
			}
			to {
				transform: scale(1);
			}
		}
		
		h1 {
			font-size: 28px;
			color: #1e293b;
			margin-bottom: 12px;
			font-weight: 700;
		}
		
		.message {
			font-size: 16px;
			color: #64748b;
			margin-bottom: 32px;
			line-height: 1.6;
		}
		
		.event-details {
			background: #f8fafc;
			border-radius: 12px;
			padding: 24px;
			margin-bottom: 32px;
			text-align: left;
		}
		
		.event-details h2 {
			font-size: 18px;
			color: #1e293b;
			margin-bottom: 16px;
			font-weight: 600;
		}
		
		.detail-row {
			display: flex;
			align-items: center;
			margin-bottom: 12px;
			font-size: 14px;
			color: #475569;
		}
		
		.detail-row:last-child {
			margin-bottom: 0;
		}
		
		.detail-label {
			font-weight: 600;
			color: #64748b;
			min-width: 100px;
		}
		
		.detail-value {
			color: #1e293b;
			flex: 1;
		}
		
		.meeting-link {
			display: inline-block;
			background: #2563eb;
			color: white;
			text-decoration: none;
			padding: 12px 24px;
			border-radius: 8px;
			font-weight: 600;
			margin-top: 8px;
			transition: background 0.2s;
		}
		
		.meeting-link:hover {
			background: #1d4ed8;
		}
		
		.close-btn {
			background: #e2e8f0;
			color: #475569;
			border: none;
			padding: 12px 24px;
			border-radius: 8px;
			font-weight: 600;
			cursor: pointer;
			transition: background 0.2s;
		}
		
		.close-btn:hover {
			background: #cbd5e1;
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="success-icon"></div>
		<h1>Lịch đã được chấp nhận!</h1>
		<p class="message">Yêu cầu đặt lịch của bạn đã được chấp nhận. Sự kiện đã được thêm vào lịch.</p>
		
		<div class="event-details">
			<h2>Chi tiết sự kiện</h2>
			<div class="detail-row">
				<span class="detail-label">Tiêu đề:</span>
				<span class="detail-value">%s</span>
			</div>
			<div class="detail-row">
				<span class="detail-label">Thời gian:</span>
				<span class="detail-value">%s</span>
			</div>
			%s
		</div>
		
		<button class="close-btn" onclick="window.close()">Đóng</button>
	</div>
</body>
</html>`,
		html.EscapeString(ev.Title),
		eventTimeStr,
		func() string {
			if ev.MeetingLink != nil && *ev.MeetingLink != "" {
				return fmt.Sprintf(`
			<div class="detail-row">
				<span class="detail-label">Meeting Link:</span>
				<span class="detail-value">
					<a href="%s" target="_blank" class="meeting-link">Tham gia Google Meet</a>
				</span>
			</div>`, html.EscapeString(*ev.MeetingLink))
			}
			return ""
		}(),
	)

	return c.HTML(http.StatusOK, html)
}

func (b *BookingController) PublicTokenDecline(c echo.Context) error {
	id := c.Param("id")
	token := c.QueryParam("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "missing token", nil))
	}
	eventID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid event id", nil))
	}
	ev, err := b.MeetingRepo.GetEventByID(c.Request().Context(), eventID)
	if err != nil || ev == nil {
		return c.JSON(http.StatusNotFound, errors.NewAppError(errors.ErrNotFound, "event not found", err))
	}
	claims, err := utils.ValidateAndParseToken(token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errors.NewAppError(errors.ErrUnauthorized, "invalid token", err))
	}
	if ev.HostID == nil || *ev.HostID != claims.UserID {
		return c.JSON(http.StatusForbidden, errors.NewAppError(errors.ErrForbidden, "not authorized", nil))
	}
	ev.Status = meetentity.EventStatusCancelled
	if err := b.MeetingRepo.UpdateEvent(c.Request().Context(), ev); err != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewAppError(errors.ErrInternalServer, "failed to update event", err))
	}
	return c.JSON(http.StatusOK, map[string]any{"message": "declined"})
}
func computeFreeSlots(start, end time.Time, busy []caldto.TimeSlot, interval int, window string) []map[string]string {
	occupied := make([][2]time.Time, 0, len(busy))
	for _, b := range busy {
		st, err1 := time.Parse(time.RFC3339, b.Start)
		et, err2 := time.Parse(time.RFC3339, b.End)
		if err1 == nil && err2 == nil {
			occupied = append(occupied, [2]time.Time{st, et})
		}
	}
	var slots []map[string]string
	step := time.Duration(interval) * time.Minute
	for t := start; t.Add(step).Before(end) || t.Add(step).Equal(end); t = t.Add(step) {
		u := t.Add(step)
		if overlaps(t, u, occupied) {
			continue
		}
		if allowWindowWithProfile(window, t, u) {
			slots = append(slots, map[string]string{
				"start": t.Format(time.RFC3339),
				"end":   u.Format(time.RFC3339),
			})
		}
	}
	return slots
}

func overlaps(st, et time.Time, occ [][2]time.Time) bool {
	for _, o := range occ {
		if st.Before(o[1]) && et.After(o[0]) {
			return true
		}
	}
	return false
}

func allowWindowWithProfile(window string, st, et time.Time) bool {
	switch strings.ToLower(strings.TrimSpace(window)) {
	case "wed_fri_afternoon":
		wd := st.Weekday()
		h := st.Hour()
		return (wd == time.Wednesday || wd == time.Friday) && h >= 13 && et.Hour() <= 17
	default:
		return true
	}
}
