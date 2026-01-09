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
	"go-api-starter/core/utils"
	authservice "go-api-starter/modules/auth/service"
	"go-api-starter/modules/calendar/dto"
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
}

func NewBookingController(cal calsvc.CalendarService, auth authservice.AuthServiceInterface, meetingRepo meetrepo.MeetingRepositoryInterface, notif *notifsvc.NotificationService) *BookingController {
	return &BookingController{CalendarService: cal, AuthService: auth, MeetingRepo: meetingRepo, NotificationSvc: notif}
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
  slots.forEach(s=>{
    var part = ''
    var st = (s.start_time||'')
    var arr = st.split('T')
    if (arr.length > 1 && arr[1]) { part = arr[1] }
    var t = part ? part.slice(0,5) : ''
    const el=document.createElement('div'); el.className='slot'; el.textContent=t
    el.onclick=()=>{selectedSlot={start:(s.start_time||''), end:(s.end_time||'')}; setActiveSlot(el)}
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
	var req dto.SuggestedSlotsRequest
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
	start, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid start_time", nil))
	}
	end, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewAppError(errors.ErrInvalidInput, "invalid end_time", nil))
	}
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
		body := "<h3>New booking request</h3><p>Guest: " + templateEscape(req.Name) + " (" + templateEscape(strings.TrimSpace(req.Email)) + ")</p><p>Time: " + start.Format(time.RFC3339) + " → " + end.Format(time.RFC3339) + "</p><p><a href=\"" + templateEscape(acceptURL) + "\">Accept</a> &nbsp;|&nbsp; <a href=\"" + templateEscape(declineURL) + "\">Decline</a></p>"
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

func templateEscape(s string) string {
	return html.EscapeString(s)
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
	req := &dto.CreateEventRequest{
		Title:       ev.Title,
		Description: "Personal booking",
		StartTime:   ev.StartDate.Format(time.RFC3339),
		EndTime:     ev.EndDate.Format(time.RFC3339),
		Timezone:    ev.Timezone,
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
		body := "<h3>Booking confirmed</h3><p>Title: " + templateEscape(ev.Title) + "</p><p>Time: " + ev.StartDate.Format(time.RFC3339) + " → " + ev.EndDate.Format(time.RFC3339) + "</p><p>Meeting link: " + templateEscape(link) + "</p>"
		_ = utils.SendEmailTLS(*conf, utils.EmailMessage{
			To:      []string{guestEmail},
			Subject: "Your meeting is confirmed",
			Body:    body,
			IsHTML:  true,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"message": "accepted", "event_id": ev.ID.String()})
}

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
	req := &dto.CreateEventRequest{
		Title:       ev.Title,
		Description: "Personal booking",
		StartTime:   ev.StartDate.Format(time.RFC3339),
		EndTime:     ev.EndDate.Format(time.RFC3339),
		Timezone:    ev.Timezone,
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
	return c.JSON(http.StatusOK, map[string]any{"message": "accepted", "event_id": ev.ID.String()})
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
func computeFreeSlots(start, end time.Time, busy []dto.TimeSlot, interval int, window string) []map[string]string {
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
