package utils

type ServerConfig struct {
	Mode             string // Development | Production
	Port             string // Server port
	Http             string // Http | Https
	FirebaseFilePath string // Firebase config path

	AppleKeyID  string // APNS Key ID
	AppleTeamID string // Apple Team ID
	APNSFileURl string // URL key path

	MapKitToken string // Apple Mapkit token
	StripeToken string // Stripe API token

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
	ClubCollection                  string
	ClubMembersCollection           string
	ClubInvitationCollection        string
	ClubApplicationCollection       string
	ClubTransactionsCollection      string
	ClubFinancialAccountsCollection string

	// Orgs
	OrgCollection                 string
	OrgInvitationCollection       string
	OrgApplicationCollection      string
	OrganizationMembersCollection string

	// Events
	EventsCollection            string
	EventLogsCollection         string
	EventViewsCollection        string
	EventTeamsCollection        string
	EventCommentsCollection     string
	EventInvitationsCollection  string
	EventParticipantsCollection string

	// Venues
	VenuesCollection       string
	VenueRequestCollection string

	// Posts
	PostCollection          string
	PostCommentsCollection  string
	PostReactionsCollection string

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

	TagsCollections  string
	SportsCollection string
}
