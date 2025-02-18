package interfaces

import (
	auth "olympsis-server/auth/service"
	club "olympsis-server/club/service"
	event "olympsis-server/event/service"
	locale "olympsis-server/locales/service"
	snap "olympsis-server/map-snapshots/service"
	org "olympsis-server/organization/service"
	post "olympsis-server/post/service"
	report "olympsis-server/report/service"
	user "olympsis-server/user/service"
	venue "olympsis-server/venue/service"
)

// Model that helps us interface with most of the services on this server
type ServerInterface struct {
	AuthService *auth.Service
	UserService *user.Service

	EventService *event.Service
	VenueService *venue.Service

	ClubService *club.Service
	OrgService  *org.Service
	PostService *post.Service

	SnapService   *snap.Service
	ReportService *report.Service
	LocaleService *locale.Service
}

func NewServerInterface(
	a *auth.Service,
	u *user.Service,
	e *event.Service,
	v *venue.Service,
	c *club.Service,
	o *org.Service,
	p *post.Service,
	s *snap.Service,
	r *report.Service,
	l *locale.Service,
) *ServerInterface {
	return &ServerInterface{
		AuthService:   a,
		UserService:   u,
		EventService:  e,
		VenueService:  v,
		ClubService:   c,
		OrgService:    o,
		PostService:   p,
		SnapService:   s,
		ReportService: r,
		LocaleService: l,
	}
}
