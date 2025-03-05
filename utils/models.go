package utils

type ServerConfig struct {
	Mode             string // Development | Production
	Port             string // Server port
	Http             string // Http | Https
	FirebaseFilePath string // Firebase config path

	MapKitToken string // Apple Mapkit token

	NotifServiceURL   string
	StorageServiceURL string

	KeyFilePath  string // TLS key file
	CertFilePath string // TLS cert file
}

type DatabaseConfig struct {
	Name     string // main database name
	Address  string // database url
	User     string
	Password string

	LocaleName string // Locale database name
}

type CollectionsConfig struct {
	AnnouncementCollection string

	// Users
	AuthCollection string
	UserCollection string

	// Clubs
	ClubCollection            string
	ClubInvitationCollection  string
	ClubApplicationCollection string

	// Orgs
	OrgCollection            string
	OrgInvitationCollection  string
	OrgApplicationCollection string

	// Events
	EventCollection           string
	EventActivityCollection   string
	EventInvitationCollection string

	// Venues
	VenueCollection        string
	VenueRequestCollection string

	// Posts
	PostCollection    string
	CommentCollection string

	// Reports
	BugReportCollection    string
	PostReportCollection   string
	VenueReportCollection  string
	EventReportCollection  string
	MemberReportCollection string

	// Logs
	AuthLogs            string
	EventLogs           string
	VenueLogs           string
	PostLogs            string
	CommentLogs         string
	OrgLogs             string
	ClubLogs            string
	ClubApplicationLogs string

	CountriesCollection     string
	AdminAreasCollection    string
	SubAdminAreasCollection string
}
